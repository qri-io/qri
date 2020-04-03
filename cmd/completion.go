package cmd

import (
	"bytes"
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/spf13/cobra"
)

// NewAutocompleteCommand creates a new `qri complete` cobra command that prints autocomplete scripts
func NewAutocompleteCommand(_ Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &AutocompleteOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh]",
		Short: "Generates shell completion scripts",
		Long: `To load completion run

source <(qri completion [bash|zsh])

To configure your bash/zsh shell to load completions for each session add to your bashrc/zshrc

# ~/.bashrc or ~/.zshrc
source <(qri completion [bash|zsh])

Alternatively you can pipe the output to a local script and
reference that as the source for faster loading
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args)
		},
		ValidArgs: []string{"bash", "zsh"},
	}

	return cmd
}

// AutocompleteOptions encapsulates completion options
type AutocompleteOptions struct {
	ioes.IOStreams
}

// Run executes the completion command
func (o *AutocompleteOptions) Run(cmd *cobra.Command, args []string) (err error) {
	if len(args) == 0 {
		return fmt.Errorf("shell not specified")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments, expected only the shell type")
	}
	if args[0] == "bash" {
		cmd.Parent().GenBashCompletion(o.Out)
	}
	if args[0] == "zsh" {
		zshHead := `# reference kubectl completion zsh

__qri_bash_source() {
	alias shopt=':'
	alias _expand=_bash_expand
	alias _complete=_bash_comp
	emulate -L sh
	setopt kshglob noshglob braceexpand

	source "$@"
}

__qri_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift

		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__qri_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}

__qri_compgen() {
	local completions w
	completions=( $(compgen "$@") ) || return $?

	# filter by given word as prefix
	while [[ "$1" = -* && "$1" != -- ]]; do
		shift
		shift
	done
	if [[ "$1" == -- ]]; then
		shift
	fi
	for w in "${completions[@]}"; do
		if [[ "${w}" = "$1"* ]]; then
			echo "${w}"
		fi
	done
}

__qri_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}

__qri_declare() {
	if [ "$1" == "-F" ]; then
		whence -w "$@"
	else
		builtin declare "$@"
	fi
}

__qri_ltrim_colon_completions()
{
	if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
		# Remove colon-word prefix from COMPREPLY items
		local colon_word=${1%${1##*:}}
		local i=${#COMPREPLY[*]}
		while [[ $((--i)) -ge 0 ]]; do
			COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
		done
	fi
}

__qri_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}

__qri_filedir() {
	local RET OLD_IFS w qw

	__debug "_filedir $@ cur=$cur"
	if [[ "$1" = \~* ]]; then
		# somehow does not work. Maybe, zsh does not call this at all
		eval echo "$1"
		return 0
	fi

	OLD_IFS="$IFS"
	IFS=$'\n'
	if [ "$1" = "-d" ]; then
		shift
		RET=( $(compgen -d) )
	else
		RET=( $(compgen -f) )
	fi
	IFS="$OLD_IFS"

	IFS="," __debug "RET=${RET[@]} len=${#RET[@]}"

	for w in ${RET[@]}; do
		if [[ ! "${w}" = "${cur}"* ]]; then
			continue
		fi
		if eval "[[ \"\${w}\" = *.$1 || -d \"\${w}\" ]]"; then
			qw="$(__qri_quote "${w}")"
			if [ -d "${w}" ]; then
				COMPREPLY+=("${qw}/")
			else
				COMPREPLY+=("${qw}")
			fi
		fi
	done
}

__qri_quote() {
    if [[ $1 == \'* || $1 == \"* ]]; then
        # Leave out first character
        printf %q "${1:1}"
    else
    	printf %q "$1"
    fi
}

autoload -U +X bashcompinit && bashcompinit

# use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
if sed --help 2>&1 | grep -q GNU; then
	LWORD='\<'
	RWORD='\>'
fi

__qri_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/_get_comp_words_by_ref "\$@"/_get_comp_words_by_ref "\$*"/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__qri_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__qri_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__qri_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__qri_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__qri_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/builtin declare/g" \
	-e "s/\\\$(type${RWORD}/\$(__qri_type/g" \
	<<'BASH_COMPLETION_EOF'
`

		zshBody := new(bytes.Buffer)
		cmd.Parent().GenBashCompletion(zshBody)

		zshTail := `
BASH_COMPLETION_EOF
}

__qri_bash_source <(__qri_convert_bash_to_zsh)
`

		o.Out.Write([]byte(zshHead))
		o.Out.Write(zshBody.Bytes())
		o.Out.Write([]byte(zshTail))
	}
	return nil
}

const (
        bash_completion_func = `
__qri_parse_list()
{
    local qri_output out
    if qri_output=$(qri list --simple 2>/dev/null); then
        out=($(echo "${qri_output}"))
        COMPREPLY=( $( compgen -W "${out[*]}" -- "$cur" ) )
    fi
}

__qri_get_datasets()
{
    __qri_parse_list
    if [[ $? -eq 0 ]]; then
        return 0
    fi
}

__qri_custom_func() {
    case ${last_command} in
        qri_get | qri_log)
            __qri_get_datasets
            return
            ;;
        *)
            ;;
    esac
}
`)
