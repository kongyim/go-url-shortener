# PostgreSQL image
resource "docker_image" "postgres" {
  name = "postgres:16"
}

# Keep the previous volume so switching to a bind mount does not delete old data.
resource "docker_volume" "postgres_data" {
  name = "go-url-shortener-postgres-data"
}

resource "docker_container" "postgres" {
  name    = "go-url-shortener-postgres"
  image   = docker_image.postgres.image_id
  restart = "unless-stopped" # 加呢行

  env = [
    "POSTGRES_USER=app",
    "POSTGRES_DB=urlshortener",
    "POSTGRES_PASSWORD=${var.postgres_password}"
  ]

  ports {
    internal = 5432
    external = 5432
  }

  networks_advanced {
    name = docker_network.app.name
  }

  volumes {
    host_path      = abspath("${path.module}/data/postgres")
    container_path = "/var/lib/postgresql/data"
  }
}
