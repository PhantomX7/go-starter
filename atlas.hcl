data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./database/migrations",
  ]
}
env "gorm" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/15/dev"
  migration {
    dir = "file://database/migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}