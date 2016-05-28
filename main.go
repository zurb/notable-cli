package main

import (
  "encoding/json"
  "errors"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "net/url"
  "os"
  "path/filepath"
  "time"

  "github.com/briandowns/spinner"
  "github.com/deiwin/interact"
  "github.com/fatih/color"
  "github.com/howeyc/gopass"
  "github.com/mitchellh/go-homedir"
  "github.com/urfave/cli"
)

var (
  platformHost = "http://notable.dev"
  version      = "0.0.8"

  authPath      string
  s             = spinner.New(spinner.CharSets[6], 100*time.Millisecond)
  checkNotEmpty = func(input string) error {
    if input == "" {
      return errors.New("Input should not be empty!")
    }
    return nil
  }
)

func check(e error) {
  if e != nil {
    panic(e)
  }
}

// EnvConfig is the global configuration object
type EnvConfig struct {
  AuthToken string `json:"token"`
}

var envConfig = EnvConfig{}

func main() {
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
      Name:    "code",
      Aliases: []string{"c"},
      Usage:   "Send site to Notable, local or live!",
      Flags: []cli.Flag{
        cli.StringFlag{
          Name:  "dest, d",
          Value: ".",
          Usage: "destination",
        },
      },
      Action: func(c *cli.Context) error {
        loadAndCheckEnv()

        runCode(c)
        return nil
      },
    },
    {
      Name:    "notebook",
      Aliases: []string{"c"},
      Usage:   "get design feedback on your images",
      Subcommands: []cli.Command{
        {
          Name:  "create",
          Usage: "create a new notebook",
          Action: func(c *cli.Context) error {
            runNotebook(c)
            return nil
          },
        },
      },
    },
    {
      Name:    "login",
      Aliases: []string{"l"},
      Usage:   "Authenticate the CLI",
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
      Name:    "logout",
      Aliases: []string{"lo"},
      Usage:   "Deauthorize this computer",
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
  tokenData := []byte(t)
  err := ioutil.WriteFile(authPath, tokenData, 0644)
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

func currentPath() string {
  dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
  if err != nil {
    log.Fatal(err)
  }
  return dir
}
