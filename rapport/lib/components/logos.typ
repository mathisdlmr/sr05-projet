#import "../config.typ": IMAGE_BOX_MAX_HEIGHT, IMAGE_BOX_MAX_WIDTH

#let logo(school-logo) = {
    //TODO : enlever le block ?
    block[
        #box(height: IMAGE_BOX_MAX_HEIGHT, width: IMAGE_BOX_MAX_WIDTH)[
            #align(start + horizon)[
            #if school-logo == none {
                image("../assets/logo_utc.svg")
            } else {
                school-logo
            }
            ]
        ]
    ]
}

#let logos(school-logo, company-logo) = {
    //TODO : enlever le block ?
    block[
        #box(height: IMAGE_BOX_MAX_HEIGHT, width: IMAGE_BOX_MAX_WIDTH)[
            #align(start + horizon)[
            #if school-logo == none {
                image("../assets/logo_utc.svg")
            } else {
                school-logo
            }
            ]
        ]
        #h(1fr)
        #box(height: IMAGE_BOX_MAX_HEIGHT, width: IMAGE_BOX_MAX_WIDTH)[
            #align(end + horizon)[
            #company-logo
            ]
        ]
    ]
}