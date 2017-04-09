# go-public-google-drive
The purpose of this program is to serve as a replacement for Google's discontinued service to direct link to public
files on Google Drive.  This was previously possible using googledrive.com, but stopped working in 2016.

This requires you have a server with go and an api key.

## TODO
* Actually check errors instead of the current hacky way of doing everything.
* Make the final HTML page look good.
* Give the option of caching files to the server. This would obviously allow for massive speed-ups

## Known problems
* The Drive api gets mad if a single user makes too many requests per second.  This program potentially makes quite
  a few, so this can become a problem.  Either errors or massive slowdowns are possible
