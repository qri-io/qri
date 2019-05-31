/*Package ds defines the qri dataset object within starlark

  outline: ds
    ds defines the qri dataset object within starlark. it's loaded by default
    in the qri runtime

    types:
      Dataset
        a qri dataset. Datasets can be either read-only or read-write. By default datasets are read-write
        methods:
          set_meta(meta dict)
            set dataset meta component
          get_meta() dict|None
            get dataset meta component
          get_structure() dict|None
            get dataset structure component if one is defined
          set_structure(structure) structure
            set dataset structure component
          get_body() dict|list|None
            get dataset body component if one is defined
          set_body(data dict|list, parse_as? string) body
            set dataset body component. set_body has only one optional argument: 'parse_as', which defaults to the
            empty string. By default qri assumes the data value provided to set_body is an iterable starlark data
            structure (tuple, set, list, dict). When parse_as is set, set_body assumes the provided body value will
            be a string of serialized structured data in the given format. valid parse_as values are "json", "csv",
            "cbor", "xlsx".
*/
package ds
