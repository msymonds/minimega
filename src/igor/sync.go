// Copyright (2018) Sandia Corporation.
// Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
// the U.S. Government retains certain rights in this software.

package main

import (
	"fmt"
	log "minilog"
	"strconv"
)

var cmdSync = &Command{
	UsageLine: "sync",
	Short:     "synchronize igor data",
	Long: `
Does an internal check to verify the integrity of the data file. Can report and attempt to clean.
SYNOPSIS
	igor sync <[-d] [-f]> [-q] WHAT
OPTIONS
	-f, -force
	    Will force sync to fix inconsistencies found in addition to reporting
	-d, -dry-run
	    Does not attempt to make any corrections, only reports inconsistencies
	-q, -quiet
	    Suppress reports, only report errors
Possible WHATs:
arista: 	reconfigure switchports for active reservations
	`,
}

var subF bool   // -f
var force bool  // -force
var subD bool   // -d
var dryRun bool // -dry-run
var subQ bool   // -q
var quiet bool  // -quiet

func init() {
	// break init cycle
	cmdSync.Run = runSync

	cmdSync.Flag.BoolVar(&subF, "f", false, "")
	cmdSync.Flag.BoolVar(&force, "force", false, "")
	cmdSync.Flag.BoolVar(&subD, "d", false, "")
	cmdSync.Flag.BoolVar(&dryRun, "dry-run", false, "")
	cmdSync.Flag.BoolVar(&subQ, "q", false, "")
	cmdSync.Flag.BoolVar(&quiet, "quiet", false, "")
}

// Gather data integrity information, report, and fix
func runSync(cmd *Command, args []string) {
	// process flags
	dryRun = (dryRun || subD)
	force = (force || subF)
	quiet = (quiet || subQ)

	if dryRun == force {
		log.Fatal("Missing or invalid flags. Please see igor sync -h, --help")
	}

	if igor.Username != "root" {
		log.Fatalln("Sync access restricted. Please use as admin.")
	}

	if len(args) != 1 {
		log.Fatalln("Invalid arguments")
	}

	switch args[0] {
	case "arista":
		syncArista()
	default:
		log.Fatalln("Invalid arguments")
	}
}

func syncArista() {
	// first get ground truth
	gt, err := networkVlan()
	if err != nil {
		log.Fatal("Unable to acquire VLAN ground truth from arista")
	}
	// TODO: probably shouldn't iteration over .M directly
	for _, r := range igor.Reservations.M {
		if !r.IsActive(igor.Now) {
			continue
		}

		if !quiet {
			// print all nodes, Igor VLANs, and arista VLANs
			for _, host := range r.Hosts {
				vlan := strconv.Itoa(r.Vlan)
				fmt.Printf("NODE %v - IGOR VLAN: %v    ARISTA VLAN: %v\n", host, vlan, gt[host])
			}
			// TODO: do we still need this?
			fmt.Printf("set switchports for %v to %v\n", r.Hosts, r.Vlan)
		} else {
			// just print what's different
			for _, host := range r.Hosts {
				vlan := strconv.Itoa(r.Vlan)
				if gt[host] != vlan {
					fmt.Printf("DISCREPANCY IN NODE %v - IGOR VLAN: %v    ARISTA VLAN: %v\n", host, vlan, gt[host])
				}
			}
		}
		if !dryRun {
			if err := networkSet(r.Hosts, r.Vlan); err != nil {
				log.Fatal("unable to set up network isolation")
			}
		}
	}
}
