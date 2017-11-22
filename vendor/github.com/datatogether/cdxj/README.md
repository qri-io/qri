# CDXJ

<!-- Repo Badges for: Github Project, Slack, License-->

[![GitHub](https://img.shields.io/badge/project-Data_Together-487b57.svg?style=flat-square)](http://github.com/datatogether)
[![Slack](https://img.shields.io/badge/slack-Archivers-b44e88.svg?style=flat-square)](https://archivers-slack.herokuapp.com/)
[![GoDoc](https://godoc.org/github.com/datatogether/cdxj?status.svg)](http://godoc.org/github.com/datatogether/cdxj)
[![License](https://img.shields.io/github/license/datatogether/cdxj.svg)](./LICENSE) 

Golang package implementing the CDXJ file format used by OpenWayback
3.0.0 (and later) to index web archive contents (notably in WARC and
ARC files) and make them searchable via a resource resolution service.
The format builds on the CDX file format originally developed by the
Internet Archive for the indexing behind the WaybackMachine. This
specification builds on it by simplifying the primary fields while
adding a flexible JSON 'block' to each record, allowing high
flexiblity in the inclusion of additional data.

## License & Copyright

Copyright (C) 2017 Data Together
This program is free software: you can redistribute it and/or modify it under
the terms of the GNU AFFERO General Public License as published by the Free Software
Foundation, version 3.0.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE.

See the [`LICENSE`](./LICENSE) file for details.

## Getting Involved

We would love involvement from more people! If you notice any errors or would like to submit changes, please see our [Contributing Guidelines](./.github/CONTRIBUTING.md). 

We use GitHub issues for [tracking bugs and feature requests](https://github.com/datatogether/cdxj/issues) and Pull Requests (PRs) for [submitting changes](https://github.com/datatogether/cdxj/pulls)

## Installation 

Use in any golang package with:

`import "github.com/datatogether/cdxj"`

## Development

Coming Soon
