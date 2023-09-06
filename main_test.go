package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"net/url"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func createBooks(t *testing.T, db *gorm.DB, count int) []*Book {
	books := []*Book{}
	t.Helper()
	for i := 0; i < count; i++ {
		b := &Book{
			Title:  fmt.Sprintf("Book%03d", i),
			Author: fmt.Sprintf("Author%03d", i),
		}
		if err := db.Create(b).Error; err != nil {
			t.Fatalf("error creating book: %s", err)
		}
		books = append(books, b)
	}
	return books
}

func bodyHasFragments(t *testing.T, body string, fragments []string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected body to contain '%s', got %s", fragment, body)
		}
	}
}

func getHasStatus(t *testing.T, db *gorm.DB, path string, status int) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	ctx, router := gin.CreateTestContext(w)
	setupRouter(router, db)

	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		t.Errorf("got error: %s", err)
	}
	router.ServeHTTP(w, req)
	if status != w.Code {
		t.Errorf("expected response code %d, got %d", status, w.Code)
	}
	return w
}
func TestDefaultRoute(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	ctx, router := gin.CreateTestContext(recorder)
	setupRouter(router, freshDb(t))

	req, err := http.NewRequestWithContext(ctx, "GET", "/books/", nil)
	if err != nil {
		t.Errorf("got error: %s", err)
	}

	router.ServeHTTP(recorder, req)
	if http.StatusOK != recorder.Code {
		t.Fatalf("expected response code %d, got %d", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()

	expected := "<h2>My Books</h2>"

	if !strings.Contains(body, expected) {
		t.Fatalf("expected response body '%s', got '%s'", expected, body)
	}
}

func TestBookIndexError(t *testing.T) {
	t.Parallel()

	db := freshDb(t)
	if err := db.Migrator().DropTable(&Book{}); err != nil {
		t.Fatalf("got error: %s", err)
	}

	_ = getHasStatus(t, db, "/books/", http.StatusInternalServerError)
}

func TestBookIndexNominal(t *testing.T) {
	t.Parallel()
	db := freshDb(t)

	books := createBooks(t, db, 2)

	w := getHasStatus(t, db, "/books/", http.StatusOK)
	body := w.Body.String()
	fragments := []string{
		"<h2>My Books</h2>",
		fmt.Sprintf("<li>%s -- %s</li>", books[0].Title, books[0].Author),
		fmt.Sprintf("<li>%s -- %s</li>", books[1].Title, books[1].Author),
	}
	bodyHasFragments(t, body, fragments)
}

func TestBookIndexTable(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name  string
		count int
	}{
		{"empty", 0},
		{"single", 1},
		{"multiple", 10},
	}

	for i := range tcs {
		tc := &tcs[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db := freshDb(t)
			books := createBooks(t, db, tc.count)

			w := getHasStatus(t, db, "/books/", http.StatusOK)
			body := w.Body.String()
			fragments := []string{
				"<h2>My Books</h2>",
			}
			for _, book := range books {
				fragments = append(fragments,
					fmt.Sprintf("<li>%s -- %s</li>",
						book.Title, book.Author))
			}
			bodyHasFragments(t, body, fragments)
		})
	}
}

func TestBookNewGet(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name string
	}{
		{"basic"},
	}

	for i := range tcs {
		tc := &tcs[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db := freshDb(t)
			w := getHasStatus(t, db, "/books/new", http.StatusOK)
			body := w.Body.String()
			fragments := []string{
				"<h2>Add a Book</h2>",
				`<form action="/books/new" method="POST">`,
				`<input type="text" name="title" id="title"`,
				`<input type="text" name="author" id="author"`,
				`<button type="submit"`,
			}
			bodyHasFragments(t, body, fragments)
		})
	}
}

func TestBookNewPost1(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name   string
		data   gin.H
		status int
	}{
		{
			"nominal",
			gin.H{"title": "my book", "author": "me"},
			http.StatusFound,
		},
	}

	for i := range tcs {
		tc := &tcs[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db := freshDb(t)
			_ = postHasStatus(t, db, "/books/new", &tc.data,
				tc.status)
		})
	}
}

func TestBookNewPost2(t *testing.T) {
	t.Parallel()

	dropTable := func(t *testing.T, db *gorm.DB) {
		err := db.Migrator().DropTable("books")
		if err != nil {
			t.Fatalf("error dropping table 'books': %s", err)
		}
	}

	tcs := []struct {
		name      string
		data      gin.H
		setup     func(*testing.T, *gorm.DB)
		status    int
		fragments []string
	}{
		{
			name:   "nominal",
			data:   gin.H{"title": "my book", "author": "me"},
			status: http.StatusFound,
		},
		{
			// This makes the field validation fail because the
			// author is empty.
			name:   "empty_author",
			data:   gin.H{"title": "1"},
			status: http.StatusBadRequest,
			fragments: []string{
				"Author is required, but was empty",
			},
		},
		{
			// This makes the field validation fail because the
			// title is empty.
			name:   "empty_title",
			data:   gin.H{"author": "9"},
			status: http.StatusBadRequest,
			fragments: []string{
				"Title is required, but was empty",
			},
		},
		{
			// This makes the field validation fail because both
			// title and author are empty.
			name:   "empty",
			data:   gin.H{},
			status: http.StatusBadRequest,
			fragments: []string{
				"Author is required, but was empty",
				"Title is required, but was empty",
			},
		},
		{
			name:   "db_error",
			data:   gin.H{"title": "a", "author": "b"},
			setup:  dropTable,
			status: http.StatusInternalServerError,
		},
	}

	for i := range tcs {
		tc := &tcs[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db := freshDb(t)

			if tc.setup != nil {
				tc.setup(t, db)
			}

			w := postHasStatus(t, db, "/books/new", &tc.data, tc.status)

			if tc.fragments != nil {
				body := w.Body.String()
				bodyHasFragments(t, body, tc.fragments)
			}

			if tc.status == http.StatusFound {
				// Make sure the record is in the db.
				books := []Book{}
				result := db.Find(&books)

				if result.Error != nil {
					t.Fatalf("error fetching books: %s", result.Error)
				}
				if result.RowsAffected != 1 {
					t.Fatalf("expected 1 row affected, got %d",
						result.RowsAffected)
				}
				if tc.data["title"] != books[0].Title {
					t.Fatalf("expected title '%s', got '%s",
						tc.data["title"], books[0].Title)
				}
				if tc.data["author"] != books[0].Author {
					t.Fatalf("expected author '%s', got '%s",
						tc.data["author"], books[0].Author)
				}

				// Check the redirect location.
				url, err := w.Result().Location()
				if err != nil {
					t.Fatalf("location check error: %s", err)
				}

				if "/books/" != url.String() {
					t.Errorf("expected location '/books/', got '%s'",
						url.String())
				}
			}
		})
	}

}

func postHasStatus(t *testing.T, db *gorm.DB, path string,
	h *gin.H, status int) *httptest.ResponseRecorder {

	t.Helper()
	data := url.Values{}
	for k, vi := range *h {
		v := vi.(string)
		data.Set(k, v)
	}

	w := httptest.NewRecorder()
	ctx, router := gin.CreateTestContext(w)
	setupRouter(router, db)

	req, err := http.NewRequestWithContext(ctx, "POST", path,
		strings.NewReader(data.Encode()))
	if err != nil {
		t.Errorf("got error: %s", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	responseHasCode(t, w, status)
	return w
}

func responseHasCode(t *testing.T, w *httptest.ResponseRecorder,
	expected int) {

	if expected != w.Code {
		t.Errorf("expected response code %d, got %d", expected, w.Code)
	}
}
