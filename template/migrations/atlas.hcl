data "external_schema" "gorm" {
  program = [
    "go", "run", "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "../internal/data/dal/model",
    "--dialect", "postgres",
  ]
}

env "local" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/15/projecttemplate?search_path=public"
  url = getenv("DATABASE_URL")

  migration {
    dir    = "file://."
    format = atlas
  }
}

env "dev" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/15/projecttemplate?search_path=public"
  url = getenv("DATABASE_URL")

  migration {
    dir = "file://."
  }
}

env "prod" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/15/projecttemplate?search_path=public"
  url = getenv("DATABASE_URL")

  migration {
    dir = "file://."
  }
}
