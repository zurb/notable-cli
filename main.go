package main

import (
  "bytes"
  "bufio"
  "fmt"
  "os"
  "io"
  "time"
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
  "github.com/mitchellh/go-homedir"
  "github.com/briandowns/spinner"
  "github.com/deiwin/interact"
  "github.com/jhoonb/archivex"
  "github.com/satori/go.uuid"
  "github.com/howeyc/gopass"
  "github.com/fatih/color"
  "github.com/urfave/cli"
)

var (
  platformHost = "https://notable.zurb.com"
  codeHost = "https://code.zurb.com"
  version = "0.0.8"
  checkNotEmpty = func(input string) error {
    if input == "" {
      return errors.New("Input should not be empty!")
    }
    return nil
  }
  captureDirectoryPrefix = "notable-captures"
  captureDirectory string
  authPath string
  s = spinner.New(spinner.CharSets[6], 100*time.Millisecond)
)

func check(e error) {
  if e != nil {
    panic(e)
  }
}

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
  url := fmt.Sprintf("%s/api/cli/sites", codeHost)
  directoryID := fmt.Sprintf("%s", uuid.NewV4())
  captureDirectory = fmt.Sprintf("%s-%s", captureDirectoryPrefix, directoryID)
  authRoot, err := homedir.Dir()
  if err != nil {
    color.Red("Cannot access your home directory to check for authentication.")
    os.Exit(1)
  }
  authPath = fmt.Sprintf("%s/.notable_auth", authRoot)
  app := cli.NewApp()
  app.EnableBashCompletion = true
  app.Name = "notable"
  app.Usage = "Interface with Notable (http://zurb.com/notable)"
  app.Version = version
  app.Author = "Jordan Humphreys (jordan@zurb.com)"
  app.Copyright = "ZURB, Inc. 2016 (http://zurb.com)"

  app.Commands = []cli.Command{
    {
      Name:      "code",
      Aliases:   []string{"c"},
      Usage:     "Send site to Notable, local or live!",
      Flags: []cli.Flag {
        cli.StringFlag{
          Name: "dest, d",
          Value: ".",
          Usage: "destination",
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

        if len(config.Url) == 0 {
          color.Red("Code requires a url to capture, example:")
          color.White(fmt.Sprintf("%s code zurb.com", os.Args[0]))
          os.Exit(1)
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

        fmt.Printf("Please enter your Notable password: ")
        password, err := gopass.GetPasswdMasked()
        if err != nil {
          log.Fatal(err)
        }
        fetchToken(email, string(password))
        return nil
      },
    },
    {
      Name:      "logout",
      Aliases:   []string{"lo"},
      Usage:     "Deauthorize this computer",
      Action: func(c *cli.Context) error {
        removeAuth()
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
  err := ioutil.WriteFile(authPath, token_data, 0644)
  check(err)
  color.Green("You are now authenticated with Notable!")
}

func removeAuth() {
  err := os.Remove(authPath)

  if err != nil {
    fmt.Println(err)
    return
  }

  color.Green("Signed out successfully!")
}

func readAuth() (EnvConfig, error) {
  file, err := ioutil.ReadFile(authPath)

  if err != nil {
    color.Red("You are not authenticated! Please run:")
    color.Green("%s login", os.Args[0])
    os.Exit(1)
  }

  return EnvConfig{
    AuthToken: string(file),
  }, nil
}

func fetchToken(e string, p string) {
  endpoint := fmt.Sprintf("%s/api/v5/platform_users/auth_cli", platformHost)
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
  if err != nil {
    fmt.Printf("ERROR: %s", err)
    os.Exit(1)
  }

  if len(envConfig.AuthToken) != 0 {
    writeAuth(envConfig.AuthToken)
  } else {
    color.Red("Invalid credentials! Try again.")
    os.Exit(1)
  }
}

func fetch(c CaptureConfig) {
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

func zip(config CaptureConfig) {
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

func upload(config CaptureConfig, url string) {
  s.Suffix = " Upload: running..."
  s.Start()
  path := fmt.Sprintf("%s/%s/%s.zip", currentPath(), captureDirectory, config.ID)
  post(path, config, url)
}

func wGetCheck() {
  _, err := exec.LookPath("wget")
  if err != nil {
    color.Red("Missing dependency!\n")
    color.Red("Please install wget using Homebrew or some other fancy way:\n")
    color.Green("brew up && brew install wget\n")
    os.Exit(1)
  }
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
  content_type := body_writer.FormDataContentType()

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
    fmt.Println("Error happened reading the body", err)
    return
  }

  var data map[string]interface{}
  if err := json.Unmarshal(body, &data); err != nil {
    panic(err)
  }

  os.Remove(path)
  os.Remove(fmt.Sprintf("%s/%s/", currentPath(), captureDirectory))

  s.Stop()
  color.Cyan("✓ Upload: complete!\n\n")
  color.Cyan("Done! Go give feedback!")
  responseUrl := data["url"].(string)
  color.Magenta(responseUrl)

  open.Run(responseUrl)
}
