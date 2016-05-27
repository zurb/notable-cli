package main

import (
  "bytes"
  "bufio"
  "fmt"
  "os"
  "io"
  "errors"
  "io/ioutil"
  "log"
  "net/url"
  "os/exec"
  "net/http"
  "mime/multipart"
  "path/filepath"
  "encoding/json"

  "github.com/skratchdot/open-golang/open"
  "github.com/deiwin/interact"
  "github.com/jhoonb/archivex"
  "github.com/satori/go.uuid"
  "github.com/fatih/color"
  "github.com/urfave/cli"
)


func check(e error) {
  if e != nil {
    panic(e)
  }
}

// First let's declare some simple input validators
var (
  checkNotEmpty = func(input string) error {
    // note that the inputs provided to these checks are already trimmed
    if input == "" {
      return errors.New("Input should not be empty!")
    }
    return nil
  }
)

type CaptureConfig struct {
  ID        string
  Recursive string
  Url       string
  Agent     string
  Path      string
  AuthToken string
}

// Configuration is the global configuration object
type EnvConfig struct {
  AuthToken      string `json:"token"`
}

var envConfig = EnvConfig{}

func main() {
  url := "http://code.notable.dev/api/cli/sites"
  app := cli.NewApp()
  app.EnableBashCompletion = true
  app.Name = "notable"
  app.Usage = "Interface with Notable"

  app.Commands = []cli.Command{
    {
      Name:      "code",
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
        loadAndCheckEnv()

        id := fmt.Sprintf("%s", uuid.NewV4())
        config := CaptureConfig{
          Recursive: "false",
          Agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.116 Safari/537.36",
          Url: c.Args().First(),
          Path: c.String("dest"),
          ID: id,
        }

        fetch(config)
        zip(config)
        upload(config, url)
        return nil
      },
    },
    {
      Name:      "login",
      Aliases:   []string{"l"},
      Usage:     "Authenticate the CLI",
      Action: func(c *cli.Context) error {
        actor := interact.NewActor(os.Stdin, os.Stdout)
        message := "Please enter your Notable email address"
        email, err := actor.PromptAndRetry(message, checkNotEmpty)
        if err != nil {
          log.Fatal(err)
        }
        message = "Please enter your Notable password"
        password, err := actor.PromptAndRetry(message, checkNotEmpty)
        if err != nil {
          log.Fatal(err)
        }
        // TODO: post login and get auth token
        fetchToken(email, password)
        return nil
      },
    },
  }

  app.Run(os.Args)
}

func loadAndCheckEnv() {
  config, err := readAuth()

  if err != nil {
    log.Fatal(err)
  }

  envConfig = config
}

func writeAuth(t string) {
  token_data := []byte(t)
  err := ioutil.WriteFile("._auth", token_data, 0644)
  check(err)
}

func readAuth() (EnvConfig, error) {
  file, err := ioutil.ReadFile("._auth")

  if err != nil {
    return EnvConfig{}, err
  }

  return EnvConfig{
    AuthToken: string(file),
  }, nil
}

func fetchToken(e string, p string) {
  endpoint := "http://notable.dev/api/v5/platform_users/auth_cli"
  v := url.Values{}
  v.Set("email", e)
  v.Add("password", p)
  var err error

  resp, err := http.PostForm(endpoint, v)
  if nil != err {
    panic(err.Error())
  }

  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)

  err = json.Unmarshal(body, &envConfig)
  writeAuth(envConfig.AuthToken)
}

func fetch(c CaptureConfig) {
  wGetCheck()
  color.Cyan("Capturing url...\n")
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
    fmt.Sprintf("--directory-prefix=captures/%s", c.ID),
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
    text := fmt.Sprintf("%s", err)
    if text == "exit status 4" {
      fmt.Fprintln(os.Stderr, "The URL you specified is not accessible.")
      os.Exit(1)
    }
  }

}

func zip(config CaptureConfig) {
  color.Cyan("Compressing...\n")
  path := fmt.Sprintf("captures/%s", config.ID)
  zip := new(archivex.ZipFile)
  zip.Create(path)
  zip.AddAll(path, true)
  zip.Close()
}

func upload(config CaptureConfig, url string) {
  color.Cyan("Uploading...\n")
  path := fmt.Sprintf("%s/captures/%s.zip", currentPath(), config.ID)
  post(path, config, url)
}

func wGetCheck() {
  _, err := exec.LookPath("wget")
  if err != nil {
    log.Fatal("\n\n[ERROR] Please install wget using Homebrew:\nbrew up && brew install wget")
  }
  // fmt.Printf("\nwget is available at %s\n", path)
}

func currentPath() string {
  dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
  if err != nil {
    log.Fatal(err)
  }
  return dir
}

func post(path string, config CaptureConfig, url string){
  file, err := os.Open(path)
  if err != nil {
    log.Fatal(err)
  }
  defer file.Close()

  /* Create a buffer to hold this multi-part form */
  body_buf := bytes.NewBufferString("")
  body_writer := multipart.NewWriter(body_buf)
  // boundary := body_writer.Boundary()
  // fmt.Println(boundary)
  content_type := body_writer.FormDataContentType()
  // fmt.Println(content_type)

  /* Create a Form Field in a simpler way */
  body_writer.WriteField("name", config.Url)
  body_writer.WriteField("token", envConfig.AuthToken)

  /* Create a completely custom Form Part (or in this case, a file) */
  // http://golang.org/src/pkg/mime/multipart/writer.go?s=2274:2352#L86
  part, err := body_writer.CreateFormFile("upload", filepath.Base(path))
  if err != nil {
    log.Fatal(err)
  }
  _, err = io.Copy(part, file)

  /* Close the body and send the request */
  body_writer.Close()
  resp, err := http.Post(url, content_type, body_buf)
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

  var data map[string]interface{}
  if err := json.Unmarshal(body, &data); err != nil {
    panic(err)
  }

  color.Green("Done! Go give feedback!")

  open.Run(data["url"].(string))
}
