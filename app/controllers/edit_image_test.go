package controllers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/m-butterfield/mattbutterfield.com/app/data"
	"github.com/m-butterfield/mattbutterfield.com/app/lib"
)

func TestEditImage(t *testing.T) {
	imageID := "test.jpg"
	ds = &testStore{
		getImage: func(id string) (*data.Image, error) {
			return &data.Image{
				ID:        imageID,
				PreviewID: "preview.jpg",
				Caption:   "test caption",
				Width:     100,
				Height:    200,
			}, nil
		},
		getAllTags: func() ([]*data.Tag, error) {
			return []*data.Tag{}, nil
		},
	}

	r, err := http.NewRequest(http.MethodGet, "/admin/edit_image/"+encodeImageID(imageID), nil)
	if err != nil {
		t.Fatal(err)
	}
	r.AddCookie(&http.Cookie{Name: "auth", Value: "1234"})
	authArray = []byte("1234")

	w := httptest.NewRecorder()
	testRouter().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected return code: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `src="`+lib.ImagesBaseURL+`preview.jpg"`) {
		t.Error("Edit page must display the preview")
	}
	if !strings.Contains(w.Body.String(), `id="image-id" value="`+encodeImageID(imageID)+`"`) {
		t.Error("Edit form must use the original ID")
	}
}

func TestDeleteImage(t *testing.T) {
	ds = &testStore{
		deleteImage: func(id string) error {
			return nil
		},
	}

	body := strings.NewReader(`{"imageID":"` + encodeImageID("test.jpg") + `"}`)
	r, err := http.NewRequest(http.MethodPost, "/admin/delete_image", body)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: "auth", Value: "1234"})
	authArray = []byte("1234")

	w := httptest.NewRecorder()
	testRouter().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected return code: %d", w.Code)
	}
}

func TestEditImageNotFound(t *testing.T) {
	ds = &testStore{
		getImage: func(id string) (*data.Image, error) {
			return nil, sql.ErrNoRows
		},
	}

	r, err := http.NewRequest(http.MethodGet, "/admin/edit_image/"+encodeImageID("nonexistent.jpg"), nil)
	if err != nil {
		t.Fatal(err)
	}
	r.AddCookie(&http.Cookie{Name: "auth", Value: "1234"})
	authArray = []byte("1234")

	w := httptest.NewRecorder()
	testRouter().ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Unexpected return code: %d", w.Code)
	}
}
