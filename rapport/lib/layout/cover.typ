#import "../components/logos.typ": logos, logo
#import "../components/title-block.typ": title-block
#import "../components/credits.typ": credits-block, credits-block-project

#let make-cover(..args) = {
    let args = args.named()
    set text(size: 13pt)
    logos(args.school-logo, args.company-logo)
    title-block(args.title, args.subtitle, args.date)
    credits-block(
        args.mentors,
        args.company,
        args.jury,
        args.author,
        args.branch,
        args.defense-date,
        args.dict,
    )
    if (args.cover-image != none) [
        #align(center)[
            #args.cover-image
        ]
    ]
    align(center + bottom)[
        #if args.defense-date != none and args.jury != none and args.jury.len() > 0 {
            [*#args.dict.defended_on_pre_date #args.defense-date #args.dict.defended_on_post_date:*]
            // Jury
            align(center)[
                #for prof in args.jury {
                [#prof #linebreak()]
                }
            ]
            v(60pt)
        }
        #if args.author != none {
            args.author
            linebreak()
        }
        #if args.semester != none {
            [#args.dict.semester #args.semester]
        }
    ]
}

#let make-cover-project(..args) = {
    let args = args.named()
    set text(size: 13pt)
    logo(args.school-logo)
    title-block(args.title, args.subtitle, args.date)
    credits-block-project(
        args.mentors,
        args.company,
        args.jury,
        args.author,
        args.branch,
        args.defense-date,
        args.dict,
    )
    if (args.cover-image != none) [
        #align(center)[
            #args.cover-image
        ]
    ]
    align(center + bottom)[
        #if args.semester != none {
            [#args.dict.semester #args.semester]
        }
    ]
}

