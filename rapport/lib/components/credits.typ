#let credits-block-project(mentors, company, jury, authors, branch, defense-date, dict) = {

    align(center)[
    
    #linebreak()
        #if authors != none {
                authors.join(", ", last: " et ")
        }
    ]

}


#let credits-block(mentors, company, jury, author, branch, semester, defense-date, dict) = {
    // TODO : enlever box ?
    box()
    h(1fr)
    grid(
        columns: (auto, 1fr, auto),
        [
            // School
            #if branch != none {[
                *#dict.school_name* \
                #branch
            ]}

            #if mentors != none and mentors.len() != 0 {
                [
                #v(10mm)
                *#dict.school_mentor*\
                #mentors.last()
                ]
            }
        ],
        [
            #set align(right)
            // Company
            #if company != none {
                align(top + right)[
                #company
                ]
            }
            
            #if mentors != none and mentors.len() != 0 { 
                [
                #v(10mm)
                *#dict.company_mentor*\
                #mentors.first()
                ]
            }
        ],
    )

}