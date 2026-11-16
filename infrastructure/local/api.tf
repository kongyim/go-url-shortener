# Read the image already built on the local Docker host.
data "docker_image" "api" {
  name = "go-url-shortener:local"
}

resource "docker_container" "api" {
  name    = "go-url-shortener-api"
  image   = data.docker_image.api.id
  restart = "unless-stopped"

  ports {
    internal = 8080
    external = 8080
  }

  networks_advanced {
    name = docker_network.app.name
  }

  env = [
    "DATABASE_URL=postgres://app:${urlencode(var.postgres_password)}@${docker_container.postgres.name}:5432/urlshortener?sslmode=disable",
    "REDIS_ADDR=${docker_container.redis.name}:6379",
  ]
}
