package main

import (
  "bytes"
  "bufio"
  "fmt"
  "os"
  "io"
  "io/ioutil"
  "log"
  "os/exec"
  "net/http"
  "mime/multipart"
  "path/filepath"

  "github.com/jhoonb/archivex"
  "github.com/satori/go.uuid"
  "github.com/urfave/cli"
)

type CaptureConfig struct {
  ID        string
  Recursive string
  Url       string
  Agent     string
  Path      string
  AuthToken string
}

func main() {
  url := "http://code.notable.dev/api/cli/sites"
  app := cli.NewApp()
  app.EnableBashCompletion = true
  app.Name = "notable"
  app.Usage = "Interface with Notable"

  // app.Flags = []cli.Flag {
  //   cli.StringFlag{
  //     Name: "dest, d",
  //     Value: ".",
  //     Usage: "destination",
  //   },
  //   cli.StringFlag{
  //     Name: "token, t",
  //     Value: "empty",
  //     Usage: "User Auth Token",
  //   },
  // }

  app.Commands = []cli.Command{
    {
      Name:      "capture",
      Aliases:   []string{"c"},
      Usage:     "Send local site to Notable",
      Flags: []cli.Flag {
        cli.StringFlag{
          Name: "dest, d",
          Value: ".",
          Usage: "destination",
        },
        cli.StringFlag{
          Name: "token, t",
          Value: "empty",
          Usage: "User Auth Token",
        },
      },
      Action: func(c *cli.Context) error {
        id := fmt.Sprintf("%s", uuid.NewV4())
        config := CaptureConfig{
          Recursive: "false",
          Agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.116 Safari/537.36",
          Url: c.Args().First(),
          Path: c.String("dest"),
          AuthToken: c.String("token"),
          ID: id,
        }

        fmt.Printf("Directory ID: %s", config.ID)
        fetch(config)
        zip(config.ID)
        upload(config, url)
        return nil
      },
    },
  }

  app.Action = func(c *cli.Context) error {
    fmt.Println("specify command")

    return nil
  }

  app.Run(os.Args)
}

func fetch(c CaptureConfig) {
  wGetCheck()
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
    fmt.Sprintf("--directory-prefix=%s", c.ID),
    c.Url,
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
    // os.Exit(1)
  }

  err = cmd.Wait()
  if err != nil {
    fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
    // os.Exit(1)
  }

}

func zip(id string) {
  fmt.Printf("Source directory: %s", id)
  zip := new(archivex.ZipFile)
  zip.Create(id)
  zip.AddAll(id, true)
  zip.Close()
}

func upload(config CaptureConfig, url string) {
  path := fmt.Sprintf("%s/%s.zip", currentPath(), config.ID)
  post(path, config)
}

func wGetCheck() {
  path, err := exec.LookPath("wget")
  if err != nil {
    log.Fatal("\n\n[ERROR] Please install wget using Homebrew:\nbrew up && brew install wget")
  }
  fmt.Printf("wget is available at %s\n", path)
}

func currentPath() string {
  dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
  if err != nil {
    log.Fatal(err)
  }
  return dir
}


func post(path string, config CaptureConfig){
  file, err := os.Open(path)
  if err != nil {
    log.Fatal(err)
  }
  defer file.Close()
  target := "http://code.notable.dev/api/cli/sites"
  //target := "http://foobar3000.com/echo"


  /* Create a buffer to hold this multi-part form */
  body_buf := bytes.NewBufferString("")
  body_writer := multipart.NewWriter(body_buf)
  boundary := body_writer.Boundary()
  fmt.Println(boundary)
  content_type := body_writer.FormDataContentType()
  fmt.Println(content_type)


  /* Create a Form Field in a simpler way */
  body_writer.WriteField("name", "New Site")
  body_writer.WriteField("description", "This is the stuff that dreams are made of.")
  body_writer.WriteField("current_platform_user_token", config.AuthToken)


  /* Create a completely custom Form Part (or in this case, a file) */
  // http://golang.org/src/pkg/mime/multipart/writer.go?s=2274:2352#L86
  part, err := body_writer.CreateFormFile("upload", filepath.Base(path))
  if err != nil {
    log.Fatal(err)
  }
  _, err = io.Copy(part, file)


  /* Close the body and send the request */
  body_writer.Close()
  resp, err := http.Post(target, content_type, body_buf)
  if nil != err {
    panic(err.Error())
  }


  /* Handle the response */
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)

  if nil != err {
    fmt.Println("errorination happened reading the body", err)
    return
  }

  fmt.Println(string(body[:]))
}
