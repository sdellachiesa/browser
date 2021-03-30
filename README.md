# Data Browser Matsch | Mazia

[![DOI](https://zenodo.org/badge/334079676.svg)](https://zenodo.org/badge/latestdoi/334079676)
![GitHub](https://img.shields.io/github/license/euracresearch/browser)
![tests](https://github.com/euracresearch/browser/workflows/test/badge.svg)

## Introduction

**[Data Browser Matsch | Mazia](https://browser.lter.eurac.edu/)**  is a user-friendly web-based application to visualize and download micrometeorological and biophysical time series of the [Long-Term Socio-Ecological Research site Matsch | Mazia in South Tyrol, Italy](http://lter.eurac.edu/en/). It is designed both for the general public and researchers. The Data Browser Matsch | Mazia drop-down menus allow the user to query the InfluxDB database in the backend by selecting the measurements, time range, land use and elevation. Interactive Grafana dashboards show dynamic graphs of the time series.


- [IT25 LT(S)ER Eurac Website](http://lter.eurac.edu/it/)
- [DEIMS ID](https://deims.org/11696de6-0ab9-4c94-a06b-7ce40f56c964)
- [LTER Italy Val Mazia](http://www.lteritalia.it/?q=macrositi/it25-val-di-mazia)
- [Homepage Public Dahshboards](https://dashboard.alpenv.eurac.edu/d/pv9WwNWGk/homepage-public?orgId=1)
- [Software Description Paper](https://riojournal.com/article/63748/)

## Components

Data Browser Matsch | Mazia is a web application composed of three parts: 
1) Two backends: [InfluxDB](https://www.influxdata.com/) and [SnipeIT](https://snipeitapp.com/). 
2) A frontend written in Go that talks to the backends. 
3) A HTML/JavaScript client that implements the user interface and makes HTTP requests to the frontend.

## About this repository

This repository contains the frontend code and the HTML/JavaScript client.

## Issues 

If you encounter a bug or have a feature suggestion, please first check the [open issues](https://github.com/euracresearch/browser/issues) to see if your request is already being discussed. If an issue does not already exit, feel free to [file an issue](https://github.com/euracresearch/browser/issues/new).

## Contributing

If you would like to contribute, please first check the [open issues](https://github.com/euracresearch/browser/issues) to see if the feature or bug is already being discussed or worked on. If not please feel free to [file an issue](https://github.com/euracresearch/browser/issues/new) before sending any code.

To finally contribute fork the repository, create a dedicated feature branch containing the appropriate changes and make a Merge Request for review.

## License 

This project is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.
