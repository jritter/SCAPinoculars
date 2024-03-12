#!/bin/bash

sudo oscap xccdf eval --profile xccdf_org.ssgproject.content_profile_standard --results-arf resources/reports/fedora/arf.xml /usr/share/xml/scap/ssg/content/ssg-fedora-ds.xml