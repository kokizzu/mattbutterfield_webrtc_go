package controllers

import (
	"github.com/m-butterfield/mattbutterfield.com/app/static"
	"html/template"
	"net/http"
	"time"
)

var videoTemplatePath = append([]string{templatePath + "video.gohtml"}, baseTemplatePaths...)

func Video(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(&static.FlexFS{}, videoTemplatePath...)
	if err != nil {
		internalError(err, w)
		return
	}
	if err = tmpl.Execute(w, struct{ Year string }{Year: time.Now().Format("2006")}); err != nil {
		internalError(err, w)
		return
	}
}
