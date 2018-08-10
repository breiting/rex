// Copyright 2018 Bernhard Reitinger. All rights reserved.

// Package rex provides a library for accessing the REX API. REX is a cloud-based operating system
// for building augmented reality applications.
//
// The first thing you have to do is to register at https://rex.robotic-eyes.com for a free REX
// account.  Once you activated your account, you can simply create an API access token with a
// `ClientId` and a `ClientSecret`.
//
// This that information in your pocket, you can start building your REX-enabled application.  An
// example application is called rx (https://github.com/breiting/rx) which is a command line tool
// accessing the REX system.
package rex

// Global variables
var (
	RexBaseURL = "https://rex.robotic-eyes.com" // hostname for accessing REX cloud services
)
