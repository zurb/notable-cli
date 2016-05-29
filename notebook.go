package main

import (
  "fmt"
  "github.com/urfave/cli"
  "os"
  "path/filepath"
)

var (
  supportedExtensions = []string{".jpg", ".jpeg", ".png", ".gif"}
)

func runNotebook(c *cli.Context) {
  // get images
  images := findImages()

  fmt.Printf("%s", images)
  fmt.Println(len(images))
  // send multipart form data
  // open set in Notebooks
}

func stringInSlice(str string, list []string) bool {
  for _, v := range list {
    if v == str {
      return true
    }
  }
  return false
}

func findImages() []string {
  images := []string{}
  dirname := "." + string(filepath.Separator)

  d, err := os.Open(currentPath())
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  defer d.Close()

  files, err := d.Readdir(-1)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }

  for _, file := range files {
    if file.Mode().IsRegular() {
      ext := filepath.Ext(file.Name())
      // fmt.Printf("%s\n", ext)
      if stringInSlice(ext, supportedExtensions) {
        images = append(images, file.Name())
      }
    }
  }

  return images
}
