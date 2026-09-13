package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-butterfield/mattbutterfield.com/app/data"
)

func TestPhotos(t *testing.T) {
	getImagesCalled := 0
	expectedBefore := time.Unix(time.Now().Unix(), 0)
	expectedLimit := 20
	ds = &testStore{
		getImages: func(before time.Time, limit int) ([]*data.Image, error) {
			getImagesCalled += 1
			if before != expectedBefore {
				t.Errorf("Unexpected before: %s != %s", before, expectedBefore)
			}
			if limit != expectedLimit {
				t.Errorf("Unexpected limit: %d != %d", limit, expectedLimit)
			}
			return []*data.Image{{
				ID:       "12345",
				Caption:  "test caption",
				Location: "test location",
				Width:    100,
				Height:   200,
			}}, nil
		},
	}
	r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/photos?before=%d", expectedBefore.Unix()), nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	testRouter().ServeHTTP(w, r)
	if getImagesCalled != 1 {
		t.Errorf("Unexpected call count for GetImages(): %d", getImagesCalled)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Unexpected return code: %d", w.Code)
	}
}

func TestPhotoFilters(t *testing.T) {
	originalStore := ds
	t.Cleanup(func() { ds = originalStore })
	before := time.Unix(1700000000, 0)
	createdAt := before.Add(-time.Hour)
	for _, tc := range []struct {
		path   string
		active string
	}{
		{"/photos", "all"},
		{"/tag/film", "film"},
		{"/tag/digital", "digital"},
	} {
		t.Run(tc.active, func(t *testing.T) {
			calls := 0
			checkQuery := func(filter string, actualBefore time.Time, limit int) ([]*data.Image, error) {
				calls++
				if filter != tc.active || !actualBefore.Equal(before) || limit != 20 {
					t.Fatalf("Unexpected query: filter=%s before=%s limit=%d", filter, actualBefore, limit)
				}
				return []*data.Image{{ID: "filtered.jpg", Width: 100, Height: 100, CreatedAt: createdAt}}, nil
			}
			ds = &testStore{
				getImages: func(before time.Time, limit int) ([]*data.Image, error) {
					return checkQuery("all", before, limit)
				},
				getImagesByTag: func(names []string, before time.Time, limit int) ([]*data.Image, error) {
					if len(names) != 1 || names[0] != tc.active {
						t.Fatalf("Unexpected tags: %v", names)
					}
					return checkQuery(names[0], before, limit)
				},
				getTagsByNames: func(names []string) ([]*data.Tag, error) {
					if len(names) != 1 || names[0] != tc.active {
						t.Fatalf("Unexpected tags: %v", names)
					}
					return []*data.Tag{{Name: names[0]}}, nil
				},
			}
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("%s?before=%d", tc.path, before.Unix()), nil)
			w := httptest.NewRecorder()
			testRouter().ServeHTTP(w, r)
			if w.Code != http.StatusOK || calls != 1 {
				t.Fatalf("Unexpected response: status=%d query calls=%d", w.Code, calls)
			}
			body := w.Body.String()
			_, filters, found := strings.Cut(body, `<nav class="photo-filters" aria-label="Photo filters">`)
			filters, _, closed := strings.Cut(filters, "</nav>")
			if !found || !closed {
				t.Fatal("Missing photo filter navigation")
			}
			if !strings.Contains(filters, fmt.Sprintf(`<span aria-current="page">%s</span>`, tc.active)) {
				t.Errorf("Selected filter %s is not marked active", tc.active)
			}
			for _, path := range []string{"/photos", "/tag/film", "/tag/digital"} {
				hasLink := strings.Contains(filters, fmt.Sprintf(`href="%s"`, path))
				if path == tc.path && hasLink {
					t.Errorf("Active photo filter is still a link: %s", path)
				} else if path != tc.path && !hasLink {
					t.Errorf("Missing photo filter link: %s", path)
				}
			}
			nextURL := fmt.Sprintf("%s?before=%d", tc.path, createdAt.Unix())
			if !strings.Contains(body, fmt.Sprintf(`href="%s">next</a>`, nextURL)) {
				t.Errorf("Missing filtered pagination link: %s", nextURL)
			}
		})
	}
}
