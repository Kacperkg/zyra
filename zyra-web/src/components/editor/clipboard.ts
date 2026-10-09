// Only extract image addresses. Never insert external HTML or clipboard pixels.
export function clipboardImageSources(html: string): string[] {
  const template = document.createElement("template");
  template.innerHTML = html;
  return [...template.content.querySelectorAll("img")].flatMap((image) => {
    try {
      const url = new URL(image.getAttribute("src") || "");
      return url.protocol === "https:" &&
        !url.username &&
        !url.password &&
        url.href.length <= 2048
        ? [url.href]
        : [];
    } catch {
      return [];
    }
  });
}

export function validateImageURL(value: string): Promise<void> {
  if (!value) return Promise.resolve();
  let url: URL;
  try {
    url = new URL(value);
  } catch {
    return Promise.reject(new Error("Enter a direct HTTPS image URL."));
  }
  if (url.protocol !== "https:" || url.username || url.password)
    return Promise.reject(
      new Error("Enter a direct HTTPS image URL without credentials."),
    );
  if (
    (url.hostname === "tenor.com" || url.hostname === "www.tenor.com") &&
    url.pathname.includes("/view/")
  )
    return Promise.reject(
      new Error(
        "This is a Tenor webpage, not an image. Right-click the GIF and choose “Copy image address”, then paste that address here.",
      ),
    );
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.referrerPolicy = "no-referrer";
    const finish = (error?: string) => {
      clearTimeout(timer);
      image.onload = null;
      image.onerror = null;
      image.removeAttribute("src");
      if (error) reject(new Error(error));
      else resolve();
    };
    const timer = window.setTimeout(
      () =>
        finish(
          "Image did not load. Check that the address is a public, direct image URL.",
        ),
      10000,
    );
    image.onload = () => finish();
    image.onerror = () =>
      finish(
        "This address could not load as an image. Use “Copy image address”, not the webpage address.",
      );
    image.src = url.href;
  });
}
