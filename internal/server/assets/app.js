document.addEventListener("DOMContentLoaded", () => {
  wireCreateForm();
  wireCopyButtons();
});

function wireCreateForm() {
  const form = document.querySelector("#create-form");
  if (!form) {
    return;
  }
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    setCreateError("");
    const button = form.querySelector("button[type=submit]");
    setButtonBusy(button, true);
    try {
      const receipt = await createPaste(form);
      showReceipt(receipt);
      form.reset();
    } catch (error) {
      setCreateError(error.message);
    } finally {
      setButtonBusy(button, false);
    }
  });
}

async function createPaste(form) {
  const body = new URLSearchParams(new FormData(form));
  const response = await fetch(form.action, {
    method: "POST",
    headers: {
      "Accept": "application/json",
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body,
  });
  if (!response.ok) {
    throw new Error(await responseError(response));
  }
  return response.json();
}

async function responseError(response) {
  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    const body = await response.json();
    return body.error || response.statusText;
  }
  const text = await response.text();
  return text.trim() || response.statusText;
}

function showReceipt(receipt) {
  const panel = document.querySelector("#receipt");
  const pasteURL = document.querySelector("#paste-url");
  const rawURL = document.querySelector("#raw-url");
  const viewLink = document.querySelector("#view-link");
  pasteURL.value = receipt.url;
  rawURL.value = receipt.raw_url;
  viewLink.href = receipt.url;
  panel.hidden = false;
}

function setCreateError(message) {
  const error = document.querySelector("#create-error");
  if (!error) {
    return;
  }
  error.textContent = message;
  error.hidden = message === "";
}

function setButtonBusy(button, busy) {
  if (!button) {
    return;
  }
  button.disabled = busy;
  button.dataset.originalText = button.dataset.originalText || button.textContent;
  button.textContent = busy ? "Creating" : button.dataset.originalText;
}

function wireCopyButtons() {
  document.addEventListener("click", async (event) => {
    const button = event.target.closest("button");
    if (!button) {
      return;
    }
    try {
      const text = await copyTextFor(button);
      if (text === null) {
        return;
      }
      await writeClipboard(text);
      flashButton(button);
    } catch {
      flashButton(button, "Copy failed");
    }
  });
}

async function copyTextFor(button) {
  const inputID = button.getAttribute("data-copy-input");
  if (inputID) {
    return document.getElementById(inputID).value;
  }
  const directValue = button.getAttribute("data-copy-value");
  if (directValue) {
    return directValue;
  }
  const fetchURL = button.getAttribute("data-copy-fetch");
  if (fetchURL) {
    const response = await fetch(fetchURL, { headers: { "Accept": "text/plain" } });
    if (!response.ok) {
      throw new Error("copy fetch failed");
    }
    return response.text();
  }
  return null;
}

async function writeClipboard(text) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const scratch = document.createElement("textarea");
  scratch.value = text;
  scratch.setAttribute("readonly", "");
  scratch.style.position = "fixed";
  scratch.style.left = "-9999px";
  document.body.appendChild(scratch);
  scratch.select();
  document.execCommand("copy");
  scratch.remove();
}

function flashButton(button, message = "Copied") {
  const original = button.dataset.originalText || button.textContent;
  button.dataset.originalText = original;
  button.textContent = message;
  window.setTimeout(() => {
    button.textContent = original;
  }, 1100);
}
