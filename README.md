# Data Browser Matsch | Mazia

# Introduction
**[Data Browser Matsch | Mazia](https://browser.lter.eurac.edu/)**  is a user-friendly web-based application to visualize and download micrometeorological and biophysical time series of the [Long-Term Socio-Ecological Research site Matsch | Mazia in South Tyrol, Italy](http://lter.eurac.edu/en/). It is designed both for the general public and researchers. The Data Browser Matsch | Mazia drop-down menus allow the user to query the InfluxDB database in the backend by selecting the measurements, time range, land use and elevation. Interactive Grafana dashboards show dynamic graphs of the time series.


- [IT25 LT(S)ER Eurac Website](http://lter.eurac.edu/it/)
- [DEIMS ID](https://deims.org/11696de6-0ab9-4c94-a06b-7ce40f56c964)
- [LTER Italy Val Mazia](http://www.lteritalia.it/?q=macrositi/it25-val-di-mazia)
- [Homepage Public Dahshboards](https://dashboard.alpenv.eurac.edu/d/pv9WwNWGk/homepage-public?orgId=1)

# About this repository
This repository contains the frontend code.

# Components
Data Browser Matsch | Mazia is a web application composed of three parts: 
1) Two backends: [InfluxDB](https://www.influxdata.com/) and [SnipeIT](https://snipeitapp.com/). 
2) A frontend written in Go that talks to the backends. 
3) A HTML/JavaScript client that implements the user interface and makes HTTP requests to the frontend.

# Getting help
If you encounter a clear bug, create an issue and describe it.

# Contributing
If you would like to contribute, please fork the repository make appropriate changes and make a Merge Request for review.

# Authors

* [Martin Palma](http://www.eurac.edu/it/aboutus/people/Pages/staffdetails.aspx?persId=31406) 
* [Luca Cattani](http://www.eurac.edu/it/aboutus/people/Pages/staffdetails.aspx?persId=41206) 

# Contributors

* [Stefano Della Chiesa](https://github.com/sdellachiesa) 
* [Alessandro Zandonai](http://www.eurac.edu/it/aboutus/people/Pages/staffdetails.aspx?persId=23703)
* [Giulio Genova](https://github.com/GiulioGenova) 
* [Georg Niedrist](http://www.eurac.edu/it/aboutus/people/Pages/staffdetails.aspx?persId=4543) 
* [Norbert Andreatta](http://www.eurac.edu/it/aboutus/people/Pages/staffdetails.aspx?persId=20623) 

# License

This project is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.
