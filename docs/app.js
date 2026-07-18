const shots = {
  "run-details": {
    src: "./assets/run-details.png",
    alt: "Nala Chores session detail screen with chat logs and run controls",
    title: "Session details",
    body:
      "Chat-style logs separate agent messages, tool output, errors, and runner milestones so long sessions stay readable.",
  },
  configurations: {
    src: "./assets/configurations.png",
    alt: "Nala Chores configurations screen with repository, harness, agent, and secret controls",
    title: "Saved configurations",
    body:
      "Project profiles keep credentials stable, show secret presence without exposing values, and make repeat runs fast.",
  },
  "run-session": {
    src: "./assets/run-session.png",
    alt: "Nala Chores run session screen with configuration selector and prompt field",
    title: "Run session",
    body:
      "A focused prompt surface for selecting a configuration, attaching issue context, and launching the agent.",
  },
  history: {
    src: "./assets/history.png",
    alt: "Nala Chores history screen with session list and status badges",
    title: "Run history",
    body:
      "A dense run ledger keeps every configuration's session history, phase, branch, and pull request result inspectable.",
  },
};

const image = document.querySelector("[data-shot-image]");
const title = document.querySelector("[data-shot-title]");
const body = document.querySelector("[data-shot-body]");
const tabButtons = Array.from(document.querySelectorAll("[data-shot]"));

function setShot(id) {
  const shot = shots[id];
  if (!shot) return;

  tabButtons.forEach((button) => {
    const selected = button.dataset.shot === id;
    button.classList.toggle("active", selected);
    button.setAttribute("aria-selected", String(selected));
  });

  image.src = shot.src;
  image.alt = shot.alt;
  title.textContent = shot.title;
  body.textContent = shot.body;
}

tabButtons.forEach((button) => {
  button.addEventListener("click", () => setShot(button.dataset.shot));
});

document.querySelectorAll("[data-copy-target]").forEach((button) => {
  button.addEventListener("click", async () => {
    const target = document.getElementById(button.dataset.copyTarget);
    const text = target?.innerText?.trim();
    if (!text) return;

    await navigator.clipboard.writeText(text);
    button.textContent = "Copied";
    window.setTimeout(() => {
      button.textContent = "Copy";
    }, 1400);
  });
});
