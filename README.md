# notable-cli

This is a command line interface (CLI) for [Notable](http://zurb.com/notable). Right now it supports uploading sites to Notable Code. **This currently only supports Mac OSX.**

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

You can log out by running:

```
notable logout
```

Once authenticated, you can run any of the following commands.

## Upload to Notable Code

You can capture any URL, including local by running the `code` command. For instance, if you have a local application running at `localhost:3000` then you can run the following command to upload it to Notable Code.

```
notable code localhost:3000
```

Your browser will automatically open, once captured, to the Notable Code site that you just uploaded. Sites are uploaded to the account that you have active on the website. Switching accounts on the site will change which account the site is added to.

You can capture live sites the same way:

```
notable code http://www.nytimes.com/
```

Or you can use the shorthand command:

```
notable c http://zurb.com/notable
```
