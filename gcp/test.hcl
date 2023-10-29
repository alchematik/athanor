provider "gcp" {
  name    = "gcp"
  version = "v0.0.1"
}

provider "aws" {
  name    = "aws"
  version = "v0.0.1"
}

id "gcp" "bucket" "test-bucket" {
  project  = "textapp-389501"
  region   = "us-east4"
  name     = "text-app-function-repo-test-2"
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
  /*
  config {
    contents = get.gcp.bucket
  }

  */
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

/*

id "gcp" "bucket_and_object" "test" {
  bucket_name = "my-bucket"
  object_name = "my-object"
}

create "gcp" "bucket_and_object" {
  id = id.gcp.bucket_and_object.test
  version = "v0.0.1"
  config {
    contents = "bla"
  }
}

*/
