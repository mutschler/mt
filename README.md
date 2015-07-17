# mt

mt is a lightweight golang movie thumbnailer currently still in development

at the moment mt can't be configured and will write a jpg file with 24 screenshots to the same direcetory as the source file

## Installation

`mt` uses the ffmpeg av library, so you'll need those librarys and then just run:

`go get bitbucket.org/raphaelmutschler/mt`

## Settings

Settings can be alternated via config files in JSON format, there are 3 directories in which the config can be saved:

`/etc/mt/`, `$HOME/.mt/` and the current directory

just create a file called `md.json` in any of this locations to change the settings...

alternatively you can set environment variables to change some of the settings at run time. env vars use the `MT_` prefix:

`MT_NUMCAPS=20 mt myvideo.mkv` will change the numcaps settings to 20 for this run only

## Available Config Options:

| name | default value | description |
| ---- | ----- | ----------- |
| numcaps | 4 | number of screenshots to take |
| columns | 2 | how many columns should be used |
| padding | 5 | add a padding around the images |
| width | 400 | width of a single screenshot |
| font_all | "Ubuntu.ttf" | Font to use for timestamps and header |
| font_size | 12 | font size |
| disable_timestamps | false | option to disable timestamp generation |
| timestamp_opacity | 1.0 | opacity of the timestamps must be from 0.0 to 1.0 |
| bg_content | "0,0,0" | RGB values for background color |
| single_images | false | will create a single image for each screenshot |
| header | true | append a header with file informations |
| bg_header | "0,0,0" | header background color |
| fg_header | "255,255,255" | header font color |


## Usage

just run `mt` and provide any video file as args:
`mt video.avi`

### example:

more examples can be found in the example older

![alt text](./example/mt_2x2.jpg)