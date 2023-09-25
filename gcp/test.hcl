provider "gcp" {
  name    = "gcp"
  version = "v0.0.1"
}

id "gcp" "bucket" "test-bucket" {
  account  = "something"
  region   = "us-va"
  name     = "test-bucket"
}

op "create" {
  version  = "v0.0.1"
  id       = id.gcp.bucket.test-bucket
}

id "gcp" "bucket_object" "test-object" {
  bucket   = id.gcp.bucket.test-bucket
  name     = "test/v0.0.1"
}

op "create" {
  version = "v0.0.1"
  id      = id.gcp.bucket_object.test-object
  config  = {
    contents = "bla"
  }
}

id "gcp" "resource_policy" "bucket-policy" {
  name     = "bucket-policy"
  resource = id.gcp.bucket.test-bucket
}

op "create" {
  version = "v0.0.1"
  id      = id.resource_policy.bucket-policy
}
