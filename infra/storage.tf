resource "google_storage_bucket" "files" {
  name     = "files.mattbutterfield.com"
  location = "US"

  cors {
    max_age_seconds = 3600
    method = [
      "OPTIONS",
      "GET",
      "POST",
      "PUT",
      "HEAD",
    ]
    origin = [
      "*",
    ]
    response_header = [
      "Content-Type",
      "Access-Control-Allow-Headers",
      "Access-Control-Allow-Origin",
    ]
  }

}

resource "google_storage_bucket" "images" {
  name                        = "images.mattbutterfield.com"
  location                    = "US"
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_iam_member" "images_public" {
  bucket = google_storage_bucket.images.name
  role   = "roles/storage.objectViewer"
  member = "allUsers"
}

resource "google_project_iam_member" "uploader" {
  project = var.project
  role    = "roles/storage.objectCreator"
  member  = "serviceAccount:${google_service_account.mattbutterfield_uploader.email}"
}

resource "google_project_iam_member" "storage_object_admin" {
  project = var.project
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.mattbutterfield_cloud_run.email}"
}

# start of https url forwarding to bucket content

resource "google_compute_managed_ssl_certificate" "images" {
  name = "images"

  managed {
    domains = ["images.mattbutterfield.com"]
  }
}

resource "google_compute_managed_ssl_certificate" "files" {
  name = "files"

  managed {
    domains = ["files.mattbutterfield.com"]
  }
}

resource "google_compute_global_address" "images" {
  name    = "images"
  address = "34.98.91.63"
}

resource "google_compute_global_address" "files" {
  name    = "files"
  address = "34.120.4.174"
}

resource "google_compute_backend_bucket" "images" {
  name        = "images"
  bucket_name = google_storage_bucket.images.name
}

resource "google_compute_backend_bucket" "files" {
  name        = "files"
  bucket_name = google_storage_bucket.files.name
}

resource "google_compute_url_map" "images" {
  name = "images"
  default_service = google_compute_backend_bucket.images.id
}

resource "google_compute_url_map" "files" {
  name = "files"
  default_service = google_compute_backend_bucket.files.id
}

resource "google_compute_target_https_proxy" "images" {
  name             = "images-target-proxy"
  url_map          = google_compute_url_map.images.id
  ssl_certificates = [google_compute_managed_ssl_certificate.images.id]
}

resource "google_compute_target_https_proxy" "files" {
  name             = "files-target-proxy"
  url_map          = google_compute_url_map.files.id
  ssl_certificates = [google_compute_managed_ssl_certificate.files.id]
}

resource "google_compute_global_forwarding_rule" "images" {
  name       = "images"
  target     = google_compute_target_https_proxy.images.id
  port_range = 443
}

resource "google_compute_global_forwarding_rule" "files" {
  name       = "files"
  ip_version = "IPV4"
  target     = google_compute_target_https_proxy.files.id
  port_range = 443
}
