package controllers

import (
	"database/sql"
	"github.com/m-butterfield/mattbutterfield.com/app/data"
	"github.com/m-butterfield/mattbutterfield.com/app/lib"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHome(t *testing.T) {
	imageID := lib.HomeImage
	previewID := "preview.jpg"
	randImageID := "blerp"
	getImageCalled, randomCalled := 0, 0
	ds = &testStore{
		getImage: func(id string) (*data.Image, error) {
			getImageCalled += 1
			if id != imageID {
				t.Errorf("GetImage called with unexpected image id: %s", id)
			}
			return &data.Image{ID: imageID, PreviewID: previewID}, nil
		},
		getRandomImage: func() (*data.Image, error) {
			randomCalled += 1
			return &data.Image{ID: randImageID}, nil
		},
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/img/"+encodeImageID(imageID), nil)
	testRouter().ServeHTTP(w, req)

	if getImageCalled != 1 {
		t.Errorf("Unexpected call count for GetImage(): %d", getImageCalled)
	}
	if randomCalled != 1 {
		t.Errorf("Unexpected call count for GetRandomImage(): %d", randomCalled)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Unexpected return code: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `src="`+lib.ImagesBaseURL+previewID+`"`) {
		t.Error("Image page must display the preview")
	}
}

func TestHomeOriginalImageLink(t *testing.T) {
	originalStore, originalAuth := ds, authArray
	t.Cleanup(func() { ds, authArray = originalStore, originalAuth })
	authArray = []byte("1234")
	ds = &testStore{
		getImage: func(id string) (*data.Image, error) {
			return &data.Image{ID: "original.jpg", PreviewID: "preview.jpg"}, nil
		},
		getRandomImage: func() (*data.Image, error) {
			return &data.Image{ID: "next.jpg"}, nil
		},
	}
	for _, tc := range []struct {
		name   string
		cookie string
		linked bool
	}{
		{"admin", "1234", true},
		{"anonymous", "", false},
		{"invalid auth", "wrong", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, makeImagePath("original.jpg"), nil)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "auth", Value: tc.cookie})
			}
			w := httptest.NewRecorder()
			testRouter().ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("Unexpected return code: %d", w.Code)
			}
			body := w.Body.String()
			preview := `<img src="` + lib.ImagesBaseURL + `preview.jpg"`
			if !strings.Contains(body, preview) {
				t.Error("Image page must display the preview")
			}
			_, linkContent, linked := strings.Cut(body, `<a href="`+lib.ImagesBaseURL+`original.jpg" style="display: contents">`)
			if linked != tc.linked {
				t.Errorf("Original image link present: %t; want %t", linked, tc.linked)
			}
			if linked {
				linkContent, _, closed := strings.Cut(linkContent, "</a>")
				if !closed || !strings.Contains(linkContent, preview) {
					t.Error("Original image link must wrap the preview image")
				}
			}
		})
	}
}

func TestHomeInvalidID(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/img/"+"MjAwO", nil)
	w := httptest.NewRecorder()

	testRouter().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Unexpected return code: %d", w.Code)
	}
}

func TestHomeImageNotFound(t *testing.T) {
	getImageCalled := 0
	ds = &testStore{
		getImage: func(id string) (*data.Image, error) {
			getImageCalled += 1
			return nil, sql.ErrNoRows
		},
	}

	r, _ := http.NewRequest(http.MethodGet, "/img/"+encodeImageID("1234"), nil)
	w := httptest.NewRecorder()

	testRouter().ServeHTTP(w, r)
	if getImageCalled != 1 {
		t.Errorf("Unexpected call count for GetImage(): %d", getImageCalled)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("Unexpected return code: %d", w.Code)
	}
}
