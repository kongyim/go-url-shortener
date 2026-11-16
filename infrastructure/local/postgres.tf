# PostgreSQL image
resource "docker_image" "postgres" {
  name = "postgres:16"
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
    ip       = "127.0.0.1"
  }

  networks_advanced {
    name = docker_network.app.name
  }

  volumes {
    host_path      = abspath("${path.module}/../../.local/postgres")
    container_path = "/var/lib/postgresql/data"
  }
}
