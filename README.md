
makemessage
===========

Fetch gettext-translatable strings from .go files and templates, and update PO-files with the new strings.

This tool can both identify strings in go-code, and also strings used in django-templates, using
the *trans* and *blocktrans* tags. It is intended to be used together with [pongo-trans](https://github.com/yzzyx/pongo-trans),
but can be used with other implementations as well.

Installation
------------

```
go install github.com/yzzyx/makemessage
```

Dependencies
------------

In order to update existing .po-files, this command requires that the gettext utilities are installed.

Usage
-----

```
Usage of ./makemessage:
  -l, --languages strings             languages to process
  -o, --output string                 directory to place message files in (default "locales")
  -p, --package-paths strings         paths to go packages to parse (use '.' to parse the current directory)
  -r, --recursive                     recurse into sub-packages
  -e, --template-extensions strings   extensions of template files (default [.html])
  -t, --template-paths strings        paths to template directories to parse
```

Makemessage will automatically create a 'locales'-directory (or the directory specified in the 'output'-argument)
if not found. It will also create individual directories for each language specified on the commandline.

At least one language must be specified, and either a package path or a template path.

Example
-------

Search the current folder (recursively) for .go-files containing gotext strings, updating the language 'sv_SE'
```
$ makemessage -l sv_SE -r -p .
```

Search the folder "templates" for .html-files containing trans-/blocktrans tags, updating the language 'sv_SE'
```
$ makemessage -l sv_SE -t templates
```

Both of the above, combined in one command:
```
$ makemessage -l sv_SE -t templates -r -p .
```