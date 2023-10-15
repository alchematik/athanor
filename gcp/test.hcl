provider "gcp" {
  name    = "gcp"
  version = "v0.0.1"
}

provider "aws" {
  name    = "aws"
  version = "v0.0.1"
}

id "gcp" "bucket" "test-bucket" {
  project  = "test-project"
  region   = "us-va"
  name     = "test-bucket"
}

create "gcp" "bucket" {
  version  = "v0.0.1"
  id       = id.gcp.bucket.test-bucket
}

id "gcp" "bucket_object" "test-object" {
  bucket   = id.gcp.bucket.test-bucket
  name     = "test"
}

create "gcp" "bucket_object" {
  version = "v0.0.1"
  id      = id.gcp.bucket_object.test-object
  config {
    contents = "bla"
  }
}

id "gcp" "resource_policy" "bucket-policy" {
  name          = "bucket-policy"
  resource      = id.gcp.bucket.test-bucket
}

create "gcp" "resource_policy" {
  version = "v0.0.1"
  id      = id.gcp.resource_policy.bucket-policy
}

id "aws" "bucket" "test-bucket" {
  account = "test-account"
  region  = "us-east-1"
  name    = "aws-test-bucket"
}

create "aws" "bucket" {
  version = "v0.0.1"
  id      = id.aws.bucket.test-bucket
}
