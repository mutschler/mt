# mt

mt is a lightweight golang movie thumbnailer currently still in development

at the moment mt can't be configured and will write a jpg file with 24 screenshots to the same direcetory as the source file

## Installation

`mt` uses the ffmpeg av library, so you'll need those librarys and then just run:

`go get bitbucket.org/raphaelmutschler/mt`

### example usage:

`mt BigBuckBunny_512kb.mp4`

example output can be found in example folder
