# Redis image
resource "docker_image" "redis" {
  name = "redis:7"
}

resource "docker_container" "redis" {
  name    = "go-url-shortener-redis"
  image   = docker_image.redis.image_id
  restart = "unless-stopped" # 加呢行

  ports {
    internal = 6379
    external = 6379
    ip       = "127.0.0.1"
  }

  networks_advanced {
    name = docker_network.app.name
  }
}
