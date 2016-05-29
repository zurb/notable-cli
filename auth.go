package main

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "net/url"
  "os"

  "github.com/deiwin/interact"
  "github.com/fatih/color"
  "github.com/howeyc/gopass"
  "github.com/urfave/cli"
)

func runAuth(c *cli.Context) {
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
  fetchAuth(email, string(password))
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

func fetchAuth(e string, p string) {
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
