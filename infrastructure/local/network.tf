# Private network for our local services
resource "docker_network" "app" {
  name = "go-url-shortener-network"
}
