provider {
  version = "v0.0.1"
  name    = "aws"
}

resource "bucket" {
  modifiers = ["create", "delete"]
  identifier "account" {
    type        = "string"
    description = "the account that the bucket belongs to."
  }
  identifier "region" {
    type        = "string"
    description = "the region that the bucket belongs in."
  }
  identifier "name" {
    type        = "string"
    description = "the name of the bucket."
    is_named    = true
  }
}
