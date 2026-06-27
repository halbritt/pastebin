# Pastebin

Pastebin is a text-sharing context for moving plain text between machines, shells, and browsers by turning pasted content into a retrievable link.

## Language

**Pastebin**:
A private work/team service that stores a **Paste** and gives back a **Paste URL** so the content can be retrieved from another host or browser. It is intended to be reachable only inside a trusted network boundary, not the public internet.
_Avoid_: Public paste site, clipboard, file transfer service

**Paste**:
A bounded piece of plain text submitted to the **Pastebin**. A **Paste** may originate from a web text box, standard input, or a local text file, but it is the shared text itself, not the source file.
_Avoid_: Upload, document, blob, message

**Creator**:
The person or automation context that submits a **Paste** and receives its **Paste URL**. A **Creator** is not an account identity in the Pastebin.
_Avoid_: Account, owner, authenticated user

**Immutable Paste**:
A **Paste** whose text content does not change after it is accepted. Corrections or replacements are represented by creating a new **Paste**, not editing the existing one.
_Avoid_: Editable paste, revision

**Text Source**:
The input location a **Paste** is created from, such as a web text box, standard input, or a local text file. The source is not retained as part of the **Paste** identity.
_Avoid_: Attachment, file object, upload

**Interactive Paste**:
A **Paste** created by running the **Pastebin CLI** without arguments, pasting text into the terminal, and ending input with EOF.
_Avoid_: Editor session, prompt

**Web Paste Form**:
A browser-based creation surface for submitting a **Paste** from manually entered text.
_Avoid_: Admin console, document editor

**Paste URL**:
The link returned after a **Paste** is accepted. It includes a compact, unguessable **Paste Code**, and possessing the **Paste URL** is sufficient to read the **Paste**.
_Avoid_: Download URL, permalink

**Raw Paste URL**:
A predictable URL variant for retrieving the **Raw Paste** without browser-oriented page chrome.
_Avoid_: API URL, download endpoint

**Pastebin CLI**:
The command-line tool a **Creator** uses to create a **Paste** from a file, standard input, or an interactive terminal paste, and to retrieve an existing **Paste**. Creation prints the resulting **Paste URL**.
_Avoid_: Curl recipe, upload script

**Paste Retrieval**:
The act of reading an existing **Paste** using its **Paste URL**, **Raw Paste URL**, or **Paste Code**.
_Avoid_: Download, checkout, sync

**Paste Receipt**:
The response returned after a **Paste** is accepted. A paste receipt always identifies the resulting **Paste URL** and may include additional paste metadata for automation.
_Avoid_: Upload result, API payload

**Paste Code**:
The opaque lowercase part of a **Paste URL** that identifies a **Paste**. It is meant to be copied mechanically, not guessed or memorized, and it is not distinguished by letter case.
_Avoid_: Slug, short code, title

**Expiration**:
The point after which a **Paste** is no longer retrievable through its **Paste URL**, regardless of whether its stored record has already been cleaned up. Expiration is expected behavior, not deletion failure.
_Avoid_: Archival, permanent storage

**Size Limit**:
The maximum amount of text a single **Paste** may contain. Content beyond the size limit is rejected rather than partially stored.
_Avoid_: Chunking, streaming transfer

**Raw Paste**:
The exact text body of a **Paste** as returned for command-line retrieval. A **Raw Paste** preserves the submitted UTF-8 text bytes.
_Avoid_: Normalized output, rendered page

**Paste View**:
A browser-oriented representation of a **Paste** for reading and copying. A **Paste View** may add page chrome, but it does not redefine the **Raw Paste**.
_Avoid_: Raw response, edited paste

**Paste Metadata**:
Non-content facts about a **Paste**, such as when it was created, when it expires, and how large it is. Paste metadata does not include a creator identity.
_Avoid_: Ownership record, audit profile

**Bearer Link**:
A **Paste URL** whose possession grants read access without a separate login, token, or account check.
_Avoid_: Authenticated link, shared login

**Trusted Collaborator**:
A person or automation context inside the work/team boundary that is allowed to create or read pastes according to the Pastebin's access rules.
_Avoid_: Anonymous user, customer, public visitor

**Trusted Network Boundary**:
The network reachability boundary that limits who can create pastes. Anyone who can reach the Pastebin inside this boundary may create a **Paste**.
_Avoid_: Application login, user registry, public access

## Example Dialogue

Developer: I need to move this config from host A to host B.

Operator: Create a **Paste** and use the **Paste URL** on host B.

Developer: Does the source file path matter?

Operator: No. The **Paste** is the text content; the original file is just one way to produce it.
