# REX Golang package

This repository contains a client-side implementation for accessing REX, the cloud-based augmented reality operating system.
You can find more information about REX [here](https://www.robotic-eyes.com).

## Getting REX

Just `go get` it:

```
go get github.com/breiting/rex
```

## Getting started

The first thing you have to do is to [register](https://rex.robotic-eyes.com) for a free REX account.
Once you activated your account, you can simply create an API access token with a `ClientId` and a `ClientSecret`.

This that information in your pocket, you can start building your REX-enabled application.
An example application is called [rx](https://github.com/breiting/rx) which is a command line tool accessing the REX
system.
