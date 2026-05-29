package project

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDotnetDetector(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Api.csproj"), "<Project/>")
	write(t, filepath.Join(dir, "Properties", "launchSettings.json"),
		`{"profiles":{"http":{"applicationUrl":"http://localhost:5230"}}}`)
	got := dotnetDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Port != 5230 || !strings.Contains(got[0].Command, "dotnet watch") {
		t.Fatalf("dotnet detect wrong: %+v", got)
	}
}

func TestRubyRails(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Gemfile"), "gem 'rails', '~> 7.1'\n")
	write(t, filepath.Join(dir, "bin", "dev"), "#!/bin/sh")
	got := rubyDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Command != "bin/dev" || got[0].Port != 3000 {
		t.Fatalf("rails detect wrong: %+v", got)
	}
}

func TestPHPLaravel(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "artisan"), "#!/usr/bin/env php")
	got := phpDetector{}.Detect(dir)
	if len(got) != 1 || !strings.Contains(got[0].Command, "artisan serve") || got[0].Port != 8000 {
		t.Fatalf("laravel detect wrong: %+v", got)
	}
}

func TestJavaSpringMaven(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "pom.xml"), "<project><parent><artifactId>spring-boot-starter-parent</artifactId></parent></project>")
	write(t, filepath.Join(dir, "mvnw.cmd"), "@echo off")
	got := javaDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Command != "mvnw.cmd spring-boot:run" {
		t.Fatalf("spring maven detect wrong: %+v", got)
	}
}

func TestElixirPhoenix(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "mix.exs"), `defp deps do [{:phoenix, "~> 1.7"}] end`)
	got := elixirDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Command != "mix phx.server" || got[0].Port != 4000 {
		t.Fatalf("phoenix detect wrong: %+v", got)
	}
}

func TestDenoDetector(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "deno.jsonc"), "{\n  // tasks\n  \"tasks\": { \"dev\": \"deno run -A main.ts\" }\n}")
	got := denoDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Command != "deno task dev" {
		t.Fatalf("deno detect wrong: %+v", got)
	}
}

func TestNonSpringJavaIgnored(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "pom.xml"), "<project><artifactId>plain-lib</artifactId></project>")
	if got := (javaDetector{}).Detect(dir); got != nil {
		t.Errorf("non-spring maven project must be ignored: %+v", got)
	}
}
