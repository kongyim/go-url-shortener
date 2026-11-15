package main

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

func validateURL(rawURL string) error {
	if strings.TrimSpace(rawURL) != rawURL {
		return errors.New("URL must not contain surrounding whitespace")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("invalid URL format")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL must use http or https")
	}

	hostname := parsedURL.Hostname()

	if hostname == "" || strings.ContainsAny(hostname, " \t\r\n") {
		return errors.New("URL must have a valid hostname")
	}

	if parsedURL.User != nil {
		return errors.New("URL must not contain credentials")
	}

	if parsedURL.Fragment != "" {
		return errors.New("URL must not contain a fragment")
	}

	port := parsedURL.Port()
	if port != "" {
		if _, err := net.LookupPort("tcp", port); err != nil {
			return errors.New("invalid URL port")
		}
	}

	return nil
}
