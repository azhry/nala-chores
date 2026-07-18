const shots = {
  "run-details": {
    src: "./assets/run-details.png",
    alt: "Nala Chores session detail screen showing readable chat-style run logs",
    title: "Session detail",
    body: "Chat logs separate runner milestones, agent output, tool calls, failures, and PR results.",
  },
  configurations: {
    src: "./assets/configurations.png",
    alt: "Nala Chores configurations screen with repository, harness, agent, and secret controls",
    title: "Configurations",
    body: "Saved profiles keep repo details, credentials, agent providers, models, and harness URLs together.",
  },
  "run-session": {
    src: "./assets/run-session.png",
    alt: "Nala Chores run form with configuration selector, prompt field, and Linear issue input",
    title: "Run form",
    body: "The launch surface stays focused on one configuration, one prompt, and one issue key.",
  },
  history: {
    src: "./assets/history.png",
    alt: "Nala Chores history view with run statuses, branches, and pull request links",
    title: "History",
    body: "Each configuration keeps a readable run ledger with outcomes, branches, timestamps, and PR links.",
  },
};

const topbar = document.querySelector(".topbar");
const image = document.querySelector("[data-shot-image]");
const title = document.querySelector("[data-shot-title]");
const body = document.querySelector("[data-shot-body]");
const shotButtons = Array.from(document.querySelectorAll("[data-shot]"));

function setHeaderState() {
  topbar?.setAttribute("data-elevated", String(window.scrollY > 8));
}

function setShot(id) {
  const shot = shots[id];
  if (!shot || !image || !title || !body) return;

  shotButtons.forEach((button) => {
    const selected = button.dataset.shot === id;
    button.classList.toggle("active", selected);
    button.setAttribute("aria-pressed", String(selected));
  });

  image.dataset.changing = "true";

  window.setTimeout(() => {
    image.src = shot.src;
    image.alt = shot.alt;
    title.textContent = shot.title;
    body.textContent = shot.body;
    image.dataset.changing = "false";
  }, 110);
}

shotButtons.forEach((button) => {
  button.addEventListener("click", () => setShot(button.dataset.shot));
});

document.querySelectorAll("[data-copy-target]").forEach((button) => {
  button.addEventListener("click", async () => {
    const target = document.getElementById(button.dataset.copyTarget);
    const text = target?.innerText?.trim();
    if (!text) return;

    await navigator.clipboard.writeText(text);
    const previous = button.textContent;
    button.textContent = "Copied";
    window.setTimeout(() => {
      button.textContent = previous;
    }, 1300);
  });
});

setHeaderState();
window.addEventListener("scroll", setHeaderState, { passive: true });
