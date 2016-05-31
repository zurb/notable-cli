package main

import (
  "errors"
  "fmt"
  "log"
  "os"
  "os/exec"
  "path/filepath"
  "time"

  "github.com/briandowns/spinner"
  "github.com/fatih/color"
  "github.com/mitchellh/go-homedir"
  "github.com/urfave/cli"
)

var (
  platformHost = "https://notable.zurb.com"
  version      = "0.1.2"

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
      Aliases: []string{"n"},
      Usage:   "get design feedback on your images",
      Subcommands: []cli.Command{
        {
          Name:  "create",
          Usage: "create a new notebook",
          Action: func(c *cli.Context) error {
            loadAndCheckEnv()
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
        runAuth(c)
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

func currentPath() string {
  dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
  if err != nil {
    log.Fatal(err)
  }
  return dir
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
