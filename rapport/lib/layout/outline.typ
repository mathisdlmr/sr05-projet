#let make-outline(dict) = {
    set text(size: 15pt)
    {
        show outline.entry: it => {
            if it.element.supplement == [Appendix] {none}
            else {it}
        }
        outline(depth: 2, indent: auto, title: dict.summary)
    }
}