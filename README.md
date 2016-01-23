Roll echo is an example of an endpoint associated with an application that
will only admit service requests accompanied by a token associated with
the sample application. 

Set the envionment via sourcing the setenv.sh, using the appropriate settings in setenv.sh. Especially important
is the ECHO_WHITELISTED_CLIENT_ID variable, which is the client id that the token must be associated with.
