package processor

import (
	"context"
	"fmt"
	"testing"

	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/providers"
)

func TestReducer(t *testing.T) {
	ctx := context.Background()

	testSpecificallyPrompt := "The section on status codes"

	testSpecContent := `
Internet Engineering Task Force (IETF)                  R. Fielding, Ed.
Request for Comments: 9110                                         Adobe
STD: 97                                               M. Nottingham, Ed.
Obsoletes: 2818, 7230, 7231, 7232, 7233, 7235,                    Fastly
           7538, 7615, 7694                              J. Reschke, Ed.
Updates: 3864                                                 greenbytes
Category: Standards Track                                      June 2022
ISSN: 2070-1721


                             HTTP Semantics

Abstract

   The Hypertext Transfer Protocol (HTTP) is a stateless application-
   level protocol for distributed, collaborative, hypertext information
   systems.  This document describes the overall architecture of HTTP,
   establishes common terminology, and defines aspects of the protocol
   that are shared by all versions.  In this definition are core
   protocol elements, extensibility mechanisms, and the "http" and
   "https" Uniform Resource Identifier (URI) schemes.

   This document updates RFC 3864 and obsoletes RFCs 2818, 7231, 7232,
   7233, 7235, 7538, 7615, 7694, and portions of 7230.



	15.2.2.  101 Switching Protocols

   The 101 (Switching Protocols) status code indicates that the server
   understands and is willing to comply with the client's request, via
   the Upgrade header field (Section 7.8), for a change in the
   application protocol being used on this connection.  The server MUST
   generate an Upgrade header field in the response that indicates which
   protocol(s) will be in effect after this response.

   It is assumed that the server will only agree to switch protocols
   when it is advantageous to do so.  For example, switching to a newer
   version of HTTP might be advantageous over older versions, and
   switching to a real-time, synchronous protocol might be advantageous
   when delivering resources that use such features.

15.3.  Successful 2xx

   The 2xx (Successful) class of status code indicates that the client's
   request was successfully received, understood, and accepted.

15.3.1.  200 OK

   The 200 (OK) status code indicates that the request has succeeded.
   The content sent in a 200 response depends on the request method.
   For the methods defined by this specification, the intended meaning
   of the content can be summarized as:

   +================+============================================+
   | Request Method | Response content is a representation of:   |
   +================+============================================+
   | GET            | the target resource                        |
   +----------------+--------------------------------------------+
   | HEAD           | the target resource, like GET, but without |
   |                | transferring the representation data       |
   +----------------+--------------------------------------------+
   | POST           | the status of, or results obtained from,   |
   |                | the action                                 |
   +----------------+--------------------------------------------+
   | PUT, DELETE    | the status of the action                   |
   +----------------+--------------------------------------------+
   | OPTIONS        | communication options for the target       |
   |                | resource                                   |
   +----------------+--------------------------------------------+
   | TRACE          | the request message as received by the     |
   |                | server returning the trace                 |
   +----------------+--------------------------------------------+

                               Table 6

   Aside from responses to CONNECT, a 200 response is expected to
   contain message content unless the message framing explicitly
   indicates that the content has zero length.  If some aspect of the
   request indicates a preference for no content upon success, the
   origin server ought to send a 204 (No Content) response instead.  For
   CONNECT, there is no content because the successful result is a
   tunnel, which begins immediately after the 200 response header
   section.

   A 200 response is heuristically cacheable; i.e., unless otherwise
   indicated by the method definition or explicit cache controls (see
   Section 4.2.2 of [CACHING]).

   In 200 responses to GET or HEAD, an origin server SHOULD send any
   available validator fields (Section 8.8) for the selected
   representation, with both a strong entity tag and a Last-Modified
   date being preferred.

   In 200 responses to state-changing methods, any validator fields
   (Section 8.8) sent in the response convey the current validators for
   the new representation formed as a result of successfully applying
   the request semantics.  Note that the PUT method (Section 9.3.4) has
   additional requirements that might preclude sending such validators.

15.3.2.  201 Created

   The 201 (Created) status code indicates that the request has been
   fulfilled and has resulted in one or more new resources being
   created.  The primary resource created by the request is identified
   by either a Location header field in the response or, if no Location
   header field is received, by the target URI.

   The 201 response content typically describes and links to the
   resource(s) created.  Any validator fields (Section 8.8) sent in the
   response convey the current validators for a new representation
   created by the request.  Note that the PUT method (Section 9.3.4) has
   additional requirements that might preclude sending such validators.

15.3.3.  202 Accepted

   The 202 (Accepted) status code indicates that the request has been
   accepted for processing, but the processing has not been completed.
   The request might or might not eventually be acted upon, as it might
   be disallowed when processing actually takes place.  There is no
   facility in HTTP for re-sending a status code from an asynchronous
   operation.

   The 202 response is intentionally noncommittal.  Its purpose is to
   allow a server to accept a request for some other process (perhaps a
   batch-oriented process that is only run once per day) without
   requiring that the user agent's connection to the server persist
   until the process is completed.  The representation sent with this
   response ought to describe the request's current status and point to
   (or embed) a status monitor that can provide the user with an
   estimate of when the request will be fulfilled.

15.3.4.  203 Non-Authoritative Information

   The 203 (Non-Authoritative Information) status code indicates that
   the request was successful but the enclosed content has been modified
   from that of the origin server's 200 (OK) response by a transforming
   proxy (Section 7.7).  This status code allows the proxy to notify
   recipients when a transformation has been applied, since that
   knowledge might impact later decisions regarding the content.  For
   example, future cache validation requests for the content might only
   be applicable along the same request path (through the same proxies).

   A 203 response is heuristically cacheable; i.e., unless otherwise
   indicated by the method definition or explicit cache controls (see
   Section 4.2.2 of [CACHING]).

15.3.5.  204 No Content

   The 204 (No Content) status code indicates that the server has
   successfully fulfilled the request and that there is no additional
   content to send in the response content.  Metadata in the response
   header fields refer to the target resource and its selected
   representation after the requested action was applied.

   For example, if a 204 status code is received in response to a PUT
   request and the response contains an ETag field, then the PUT was
   successful and the ETag field value contains the entity tag for the
   new representation of that target resource.

   The 204 response allows a server to indicate that the action has been
   successfully applied to the target resource, while implying that the
   user agent does not need to traverse away from its current "document
   view" (if any).  The server assumes that the user agent will provide
   some indication of the success to its user, in accord with its own
   interface, and apply any new or updated metadata in the response to
   its active representation.

   For example, a 204 status code is commonly used with document editing
   interfaces corresponding to a "save" action, such that the document
   being saved remains available to the user for editing.  It is also
   frequently used with interfaces that expect automated data transfers
   to be prevalent, such as within distributed version control systems.

   A 204 response is terminated by the end of the header section; it
   cannot contain content or trailers.

   A 204 response is heuristically cacheable; i.e., unless otherwise
   indicated by the method definition or explicit cache controls (see
   Section 4.2.2 of [CACHING]).

15.3.6.  205 Reset Content

   The 205 (Reset Content) status code indicates that the server has
   fulfilled the request and desires that the user agent reset the
   "document view", which caused the request to be sent, to its original
   state as received from the origin server.

   This response is intended to support a common data entry use case
   where the user receives content that supports data entry (a form,
   notepad, canvas, etc.), enters or manipulates data in that space,
   causes the entered data to be submitted in a request, and then the
   data entry mechanism is reset for the next entry so that the user can
   easily initiate another input action.

   Since the 205 status code implies that no additional content will be
   provided, a server MUST NOT generate content in a 205 response.`

	testConfig := &providers.Config{
		Provider:    providers.ProviderOllama,
		Model:       "nomic-embed-text",
		APIKey:      "",
		BaseURL:     "",
		Temperature: 0.0,
		MaxTokens:   8000,
	}

	testRule := config.Rule{
		Name:        "http-spec-compliant",
		Description: "Check that the correct HTTP status codes are returned",
		Enabled:     true,
		Files: config.FilePattern{
			Include: []string{"web.go"},
			Exclude: []string{},
		},
		Specs: []config.Spec{
			{
				Path:         "RFC9110.md",
				Specifically: testSpecificallyPrompt,
			},
		},
		Prompt: "",
		FailOn: "error",
	}

	client, err := providers.NewOllamaClient[ReducerQueryResponse](testConfig)
	if err != nil {
		t.Fatal(err)
	}
	reducer, err := NewReducer(300, 10, 10, client)
	if err != nil {
		t.Fatal(err)
	}
	reduced, err := reducer.Reduce(ctx, testRule, testSpecContent, testSpecificallyPrompt)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(reduced)

}
