data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./database",
  ]
}

data "composite_schema" "all" {
  schema "public" {
    url = "file://database/schema/schema.sql"
  }
  schema "public" {
    url = data.external_schema.gorm.url
  }
}

env "gorm" {
  src = data.composite_schema.all.url
  dev = "docker://postgres/15/dev"
  migration {
    dir = "file://database/migrations"
    format = "golang-migrate"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}