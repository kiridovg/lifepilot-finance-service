env "local" {
  src = "file://internal/db/schema/schema.sql"
  dev = "docker://postgres/17/dev?search_path=public"
  migration {
    dir = "file://internal/db/migrations"
  }
}
