locals {
  api_build_context = abspath("${path.module}/../..")
  api_build_files = sort(tolist(setunion(
    toset(["Dockerfile", ".dockerignore", "go.mod", "go.sum"]),
    fileset(local.api_build_context, "cmd/**"),
    fileset(local.api_build_context, "internal/**")
  )))
}

# Build the image again when its source files or build configuration change.
resource "docker_image" "api" {
  name         = "go-url-shortener:local"
  keep_locally = true

  build {
    context    = local.api_build_context
    dockerfile = "Dockerfile"
  }

  triggers = {
    source_hash = sha256(jsonencode({
      for f in local.api_build_files : f => filesha256("${local.api_build_context}/${f}")
    }))
  }
}

# 2. Create API container
resource "docker_container" "api" {
  name    = "go-url-shortener-api"
  image   = docker_image.api.image_id
  restart = "unless-stopped"

  ports {
    internal = 8080
    external = 8080
    ip       = "127.0.0.1"
  }

  networks_advanced {
    name = docker_network.app.name
  }

  env = [
    "DATABASE_URL=postgres://app:${urlencode(var.postgres_password)}@${docker_container.postgres.name}:5432/urlshortener?sslmode=disable",
    "REDIS_ADDR=${docker_container.redis.name}:6379",
  ]
}
