package main

import (
  "bufio"
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "io/ioutil"
  "log"
  "mime/multipart"
  "net/http"
  "os"
  "os/exec"
  "path/filepath"

  "github.com/fatih/color"
  "github.com/jhoonb/archivex"
  "github.com/satori/go.uuid"
  "github.com/skratchdot/open-golang/open"
  "github.com/urfave/cli"
)

var (
  codeHost               = "https://code.zurb.com"
  captureDirectoryPrefix = "notable-captures"
  captureDirectory       string
)

// CaptureConfig is the Code configuration object
type CaptureConfig struct {
  ID        string
  Recursive string
  URL       string
  Agent     string
  Path      string
  AuthToken string
}

func runCode(c *cli.Context) {
  url := fmt.Sprintf("%s/api/cli/sites", codeHost)
  directoryID := fmt.Sprintf("%s", uuid.NewV4())
  captureDirectory = fmt.Sprintf("%s-%s", captureDirectoryPrefix, directoryID)

  id := fmt.Sprintf("%s", uuid.NewV4())
  config := CaptureConfig{
    Recursive: "false",
    Agent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.116 Safari/537.36",
    URL:       c.Args().First(),
    Path:      c.String("dest"),
    ID:        id,
  }

  if len(config.URL) == 0 {
    color.Red("Code requires a url to capture, example:")
    color.White(fmt.Sprintf("%s code zurb.com", os.Args[0]))
    os.Exit(1)
  }

  fetchCode(config)
  zipCode(config)
  uploadCode(config, url)
}

func fetchCode(c CaptureConfig) {
  wGetCheck()
  s.Prefix = ""
  s.Suffix = " Capture: running..."
  s.Start()
  args := []string{
    fmt.Sprintf("-U '%s'", c.Agent),
    "--no-clobber",
    "--adjust-extension",
    "--span-hosts",
    "--page-requisites",
    "--backup-converted",
    "--html-extension",
    "--convert-links",
    "--no-parent",
    fmt.Sprintf("--directory-prefix=%s/%s", captureDirectory, c.ID),
    c.URL,
  }
  cmd := exec.Command("wget", args...)

  cmdReader, err := cmd.StdoutPipe()
  if err != nil {
    fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Notable", err)
    os.Exit(1)
  }

  scanner := bufio.NewScanner(cmdReader)

  go func() {
    for scanner.Scan() {
      fmt.Printf("Notable capture | %s\n", scanner.Text())
    }
  }()

  err = cmd.Start()
  if err != nil {
    fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
    os.Exit(1)
  }

  err = cmd.Wait()
  if err != nil {
    text := fmt.Sprintf("%s", err)
    if text == "exit status 4" {
      color.Red("The URL you specified is not accessible.")
      os.Exit(1)
    }
  }
  s.Stop()
  color.Cyan("✓ Capture: complete!\n")

}

func zipCode(config CaptureConfig) {
  s.Suffix = " Compress: running..."
  s.Start()
  path := fmt.Sprintf("%s/%s", captureDirectory, config.ID)
  zip := new(archivex.ZipFile)
  zip.Create(path)
  zip.AddAll(path, true)
  zip.Close()
  os.RemoveAll(path)
  s.Stop()
  color.Cyan("✓ Compress: complete!\n")
}

func uploadCode(config CaptureConfig, url string) {
  s.Suffix = " Upload: running..."
  s.Start()
  path := fmt.Sprintf("%s/%s/%s.zip", currentPath(), captureDirectory, config.ID)
  postCode(path, config, url)
}

func postCode(path string, config CaptureConfig, url string) {
  file, err := os.Open(path)
  if err != nil {
    log.Fatal(err)
  }
  defer file.Close()

  /* Create a buffer to hold this multi-part form */
  bodyBuf := bytes.NewBufferString("")
  bodyWriter := multipart.NewWriter(bodyBuf)
  contentType := bodyWriter.FormDataContentType()

  /* Create a Form Field in a simpler way */
  bodyWriter.WriteField("name", config.URL)
  bodyWriter.WriteField("token", envConfig.AuthToken)

  /* Create a completely custom Form Part (or in this case, a file) */
  // http://golang.org/src/pkg/mime/multipart/writer.go?s=2274:2352#L86
  part, err := bodyWriter.CreateFormFile("upload", filepath.Base(path))
  if err != nil {
    log.Fatal(err)
  }
  _, err = io.Copy(part, file)

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

  if data["error"] != nil {
    color.Red("\nYour Notable credentials are invalid, please login again:\n")
    color.White("%s login", os.Args[0])
    os.Exit(1)
  }

  os.Remove(path)
  os.Remove(fmt.Sprintf("%s/%s/", currentPath(), captureDirectory))

  s.Stop()
  color.Cyan("✓ Upload: complete!\n\n")
  color.Cyan("Done! Go give feedback!")
  responseURL := data["url"].(string)
  color.Magenta(responseURL)

  open.Run(responseURL)
}
