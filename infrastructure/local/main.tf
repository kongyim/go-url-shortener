
terraform {
  required_version = ">= 1.5.0"

  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 4.6.0"
    }
  }
}

provider "docker" {}

# Private network for our local services
resource "docker_network" "app" {
  name = "go-url-shortener-network"
}

# PostgreSQL image
resource "docker_image" "postgres" {
  name = "postgres:16"
}

# Redis image
resource "docker_image" "redis" {
  name = "redis:7"
}




variable "postgres_password" {
  type      = string
  sensitive = true
}

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
    volume_name    = docker_volume.postgres_data.name
    container_path = "/var/lib/postgresql/data"
  }
}

resource "docker_container" "redis" {
  name    = "go-url-shortener-redis"
  image   = docker_image.redis.image_id
  restart = "unless-stopped" # 加呢行

  ports {
    internal = 6379
    external = 6379
  }

  networks_advanced {
    name = docker_network.app.name
  }
}
