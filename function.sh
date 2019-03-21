#!/bin/bash

# Debug function, only need to export something to $debug and it will show debug messages
db() {
	if [ ! -z $debug ];then
		echo "DEBUG: $@" 1>&2
	fi
}
export -f db

# ssh as given username - and send any normally arguments
# check if ssh is up, and if so, ssh as the user in first argument to that host with eventually arguments.
# Takes arguments in 'sshas [username] [remotehost] [extra arguments]'
sshas() {
	db "$FUNCNAME() was called, with args: $@"
	db "Setting \$user to $1"
	local user=$1
	shift
	db "Setting \$rhost to $1"
	rhost=$1
	shift
	db "Setting \$args to $@"
	args="$@"
	db "Calling get_host()"
	if get_host $rhost; then
        db "Checking if \$args ($args) include -p"
		if echo $args | egrep -q -- ' -p ' ;then
			db "\$args do include -p, getting portnumber"
			port=$(echo $args | egrep -o -- '-p [1-9]+' | awk '{print $2}')
			db "port=$port"
		else
			db "\$args do not include -p, using standard port"
			port=22
		fi
		db "Checking if port $port is open, wait until it is"
		until nc -w 3 -z $rhost $port; do
			echo $(date +'%Y-%m-%dT%H:%M:%S') " - port $port at $rhost not open yet . . ." 1>&2
			read -t 5
		done
		db "Open, logging in:"
		do_login="ssh ${user}@${rhost} $args"
		db "$do_login"
		eval $do_login

        # Here things happen after ssh connection has been closed.
		db "Logged out"
	else
		db "get_host() returned a bad value"
		return 1
		fi
}

s() {
	sshas $USERNAME "$@"
}

# Get a hostname, sets answer to $rhost
get_host() {
	get_host_leave_func() {
		db "Leaving $1() with \$rhost set to $rhost"
	}
	rhost=$1
	rhosts=""
	db "$FUNCNAME() was called with args: $@"
	if host $rhost >/dev/null 2>&1 ;then
        if echo $rhost | grep -q '\.';then
		    db "\$rhost found in args given: $rhost"
		    get_host_leave_func $FUNCNAME
		    return
        fi
	fi
	db "args given was not an complete hostname - doing hostdb call"
    if [ -x ~/client ]; then
        rhosts=$(~/client -configfile ~/example.toml $rhost)
    else
        echo "Can't find an executible client binary. Verify setting."
        return 1
    fi
	if [ $(echo $rhosts | tr ' ' '\n' | wc -w) -eq 1 ];then
		db "Only one match: rhosts=${rhosts}"
		rhost=$(echo $rhosts | sed 's/ //')
		get_host_leave_func $FUNCNAME
		return
	elif [ $(echo $rhosts | tr ' ' '\n' | wc -w) -gt 1 ];then
		db "Multiple possible host, asking user to choose."
		PS3="There are many possible hosts, choose one: "
		select hosts in $rhosts; do
				case $hosts in
				*)
				rhost=$hosts
					if [ X"$rhost" = X"" ];then
						echo "No host was chosen, exiting" 1>&2
						db "\$rhost is not set, return from $FUNCNAME() with error code 2"
						return 2
					fi
				db "$rhost was chosen"
				return
				;;
			esac
		done
	else
		echo "Cant find host $rhost"
		return 1
	fi
	get_host_leave_func $FUNCNAME

}
export -f get_host

