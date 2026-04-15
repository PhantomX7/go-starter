# atlas.hcl

# Combined schema from static SQL + GORM
data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./database",
  ]
}

# GORM environment
env "gorm" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/15/dev?search_path=public"
  
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