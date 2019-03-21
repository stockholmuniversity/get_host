#/usr/bin/env bash

_get_host()
{
    if [ "${#COMP_WORDS[@]}" != "2" ]; then
        return
    fi
    COMPREPLY=($(compgen -W "$(~/client -configfile ~/example.toml ${COMP_WORDS[COMP_CWORD]})" "${COMP_WORDS[1]}"))
}

complete -F _get_host get_host
complete -F _get_host s
