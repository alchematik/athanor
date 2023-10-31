version = "v0.0.1"
name    = "gcp"

resource "bucket" {
  modifiers = ["create", "delete"]
  identifier "project" {
    type        = "string"
    description = "the project that the bucket belongs to"
  }
  identifier "region" {
    type        = "string"
    description = "the region that the bucket belongs in"
  }
  identifier "name" {
    type        = "string"
    description = "the name of the bucket"
    is_named    = true
  }
}

resource "bucket_object" {
  modifiers = ["create", "delete"]
	identifier "bucket" {
    type    = "identifier_oneof"
    choices = [
      "bucket"
    ]
		description = "the bucket that the object belongs to"
	}
	identifier "name" {
		type        = "string"
		is_named    = true
		description = "the name of the bucket_object"
	}
	config "contents" {
		/* type        = "file" */
		type = "string"
		immutable   = true
	  description = "the path to the file to upload"
	}
}

resource "resource_policy" {
  modifiers = ["create", "delete"]
  identifier "resource" {
    type    = "identifier_oneof"
    choices = [
      "bucket"
    ]
    description = "the resource that the policy belongs to"
  }
  identifier "name" {
    type        = "string"
    is_named    = true
    description = "the name of the resource policy"
  }
}

/*

resource "bucket_and_object" {
  modifiers = ["create", "delete"]
  identifier "bucket_name" {
    type = "string"
  }
  identifier "object_name" {
    type = "string"
  }
  config "contents" {
    type = "string"
  }
}

group "bucket_and_object" {
  modifiers = ["create", "delete"]

  identifier "bucket_project" {
    type = "string"
  }
  identifier "bucket_region" {
    type = "string"
  }
  identifier "bucket_name" {
    type = "string"
  }
  identifier "object_name" {
    type = "string"
  }
  config "contents" {
    type = "string"
  }

  resources {
    id "gcp" "bucket" "test-bucket" {
      project  = bucket_project
      region   = bucket_region
      name     = bucket_name
    }

    id "gcp" "bucket_object" "test-object" {
      bucket   = id.gcp.bucket.test-bucket
      name     = "test"
    }
  }
}


*/
