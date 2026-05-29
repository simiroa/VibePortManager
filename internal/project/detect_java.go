//ff:what Java Spring Boot 탐지 (Maven pom.xml / Gradle build.gradle)
//ff:why Spring 백엔드 범용 대응. 래퍼(mvnw/gradlew) 우선, 기본 포트 8080
package project

import (
	"path/filepath"
	"strings"
)

type javaDetector struct{}

func (javaDetector) Name() string { return "java" }

func (javaDetector) Detect(dir string) []DetectedServer {
	base := filepath.Base(dir)

	if pom := readLower(dir, "pom.xml"); strings.Contains(pom, "spring-boot") {
		mvn := "mvn"
		switch {
		case exists(filepath.Join(dir, "mvnw.cmd")):
			mvn = "mvnw.cmd"
		case exists(filepath.Join(dir, "mvnw")):
			mvn = "./mvnw"
		}
		return []DetectedServer{{Name: base + "-spring", Command: mvn + " spring-boot:run", Port: 8080, Source: "java"}}
	}

	g := readLower(dir, "build.gradle") + readLower(dir, "build.gradle.kts")
	if strings.Contains(g, "org.springframework.boot") || strings.Contains(g, "bootrun") {
		gw := "gradle"
		switch {
		case exists(filepath.Join(dir, "gradlew.bat")):
			gw = "gradlew.bat"
		case exists(filepath.Join(dir, "gradlew")):
			gw = "./gradlew"
		}
		return []DetectedServer{{Name: base + "-spring", Command: gw + " bootRun", Port: 8080, Source: "java"}}
	}
	return nil
}
