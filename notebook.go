package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/fatih/color"
  "github.com/skratchdot/open-golang/open"
  "github.com/urfave/cli"
  "io"
  "io/ioutil"
  "log"
  "mime/multipart"
  "net/http"
  "os"
  "path/filepath"
)

var (
  notebookHost        = "http://annotate.notable.dev"
  supportedExtensions = []string{".jpg", ".jpeg", ".png", ".gif"}
)

func runNotebook(c *cli.Context) {
  // get images
  images := findImages()
  // send multipart form data
  postNotebook(images)
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

func postNotebook(images []string) {
  url := fmt.Sprintf("%s/api/v1/posts", notebookHost)
  /* Create a buffer to hold this multi-part form */
  bodyBuf := bytes.NewBufferString("")
  bodyWriter := multipart.NewWriter(bodyBuf)
  contentType := bodyWriter.FormDataContentType()

  for _, name := range images {
    path := fmt.Sprintf("%s/%s", currentPath(), name)
    file, err := os.Open(path)
    if err != nil {
      log.Fatal(err)
    }
    defer file.Close()

    /* Create a completely custom Form Part (or in this case, a file) */
    // http://golang.org/src/pkg/mime/multipart/writer.go?s=2274:2352#L86
    part, err := bodyWriter.CreateFormFile("image[]", filepath.Base(path))
    if err != nil {
      log.Fatal(err)
    }
    _, err = io.Copy(part, file)
  }

  /* Create a Form Field in a simpler way */
  bodyWriter.WriteField("name", url)
  bodyWriter.WriteField("current_platform_user_token", envConfig.AuthToken)
  /* Close the body and send the request */
  bodyWriter.Close()
  resp, err := http.Post(url, contentType, bodyBuf)
  if nil != err {
    panic(err.Error())
  }

  /* Handle the response */
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)

  if nil != err {
    fmt.Println("Error happened reading the body", err)
    return
  }

  var data map[string]interface{}
  if err := json.Unmarshal(body, &data); err != nil {
    panic(err)
  }

  s.Stop()
  color.Cyan("✓ Upload: complete!\n\n")
  color.Cyan("Done! Go give feedback!")
  responseURL := data["dashboard_item_url"].(string)
  color.Magenta(responseURL)

  open.Run(responseURL)
}
