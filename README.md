# Notable CLI
This is a command line interface (CLI) for [Notable](http://zurb.com/notable), a design feedback application by [ZURB](http://zurb.com). Right now it supports uploading sites to Notable Code. **This currently only supports Mac OSX.**

## Installing
Use [Homebrew](http://brew.sh/) to install the Notable CLI.

```
brew tap zurb/notable
brew install notable
```

You are now ready to go!


## Authenticate
Before you can use the CLI, you must authenticate it with your [Notable](http://zurb.com/notable) account.

```
notable login
```

Logout by running:

```
notable logout
```

Once authenticated, you can run any of the following commands.

## Upload to Notable Code
Capture any URL, including local by running the `code` command. For instance, if you have a local application running at `localhost:3000` then run the following command to upload it to Notable Code.

```
notable code localhost:3000
```

Your browser will automatically open, once captured, to the Notable Code site that you just uploaded.

Capture live sites the same way:

```
notable code http://www.nytimes.com/
```

Or use the shorthand command:

```
notable c http://zurb.com/notable
```

## Upgrading
To get the latest changes to the Notable CLI, run the following command:

```
brew up && brew upgrade notable
```

## Compiling from source
The provided Homebrew binary is meant for being run on Mac OSX, but if you would like to run the CLI on Windows or Linux based systems, compiling from source is your answer.

The Notable CLI is built in go, so install [go](https://golang.org) then clone down this repository into your working directory.

```
go get github.com/zurb/notable-cli
```

Then build it:

```
go build -o notable
```

That's it! You now have an executable binary for your OS.
