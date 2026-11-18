const $ = (selector, root = document) => root.querySelector(selector)
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)]
const state = { cart: { items: [], total: 0 }, busy: false }
const money = value => new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', maximumFractionDigits: 0 }).format(value)
const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[char])
const cartKey = 'developer.cart.guest'
const maxCartQuantity = 10

async function api(url, method = 'GET', body) {
  const response = await fetch(url, {
    method,
    headers: body === undefined ? {} : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body)
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    const error = new Error(data.error || `Ошибка запроса (${response.status})`)
    error.field = data.field
    throw error
  }
  return data
}

function status(element, message = '', error = false) {
  if (!element) return
  element.textContent = message
  element.dataset.error = String(error)
}

let toastTimer
function toast(message) {
  let element = $('#app-toast')
  if (!element) {
    element = document.createElement('p')
    element.id = 'app-toast'
    element.className = 'app-toast'
    element.setAttribute('role', 'status')
    document.body.append(element)
  }
  element.hidden = false
  element.textContent = message
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { element.hidden = true }, 4500)
}

function readCart() {
  try {
    const data = JSON.parse(localStorage.getItem(cartKey) || '[]')
    if (!Array.isArray(data)) return []
    const items = new Map()
    for (const item of data.slice(0, 50)) {
      if (Number.isSafeInteger(item.product_id) && item.product_id > 0 && Number.isInteger(item.quantity) && item.quantity >= 1) {
        items.set(item.product_id, { product_id: item.product_id, quantity: Math.min(item.quantity, maxCartQuantity) })
      }
    }
    return [...items.values()]
  } catch {
    return []
  }
}

function compactCart() {
  return state.cart.items.map(item => ({ product_id: item.product_id, quantity: item.quantity }))
}

function cacheCart(items = compactCart()) {
  try {
    localStorage.setItem(cartKey, JSON.stringify(items))
  } catch {
    throw new Error('Разрешите localStorage в браузере, чтобы сохранять корзину')
  }
}

async function loadCart() {
  const saved = readCart()
  const products = saved.length ? await api('/api/products') : []
  state.cart.items = saved.flatMap(item => {
    const product = products.find(product => product.id === item.product_id)
    return product ? [{ ...product, product_id: product.id, quantity: item.quantity }] : []
  })
  state.cart.total = state.cart.items.reduce((total, item) => total + item.price * item.quantity, 0)
  if (saved.length) cacheCart()
  const counter = $('#cart-count')
  if (counter) counter.textContent = String(state.cart.items.reduce((total, item) => total + item.quantity, 0))
}

async function saveCart(items) {
  cacheCart(items)
  await loadCart()
  if ($('#cart-content')) renderCart()
}

async function addToCart(id) {
  const items = compactCart()
  const existing = items.find(item => item.product_id === id)
  if (existing?.quantity >= maxCartQuantity) throw new Error(`Можно добавить не более ${maxCartQuantity} единиц одной услуги`)
  if (!existing && items.length >= 50) throw new Error('Достигнут лимит корзины')
  if (existing) existing.quantity++
  else items.push({ product_id: id, quantity: 1 })
  await saveCart(items)
  toast('Услуга добавлена в корзину')
}

function renderNavigation() {
  if (!$('.header') && $('.app-page')) {
    const header = document.createElement('div')
    header.className = 'ruled'
    header.innerHTML = `<header class="header container">
      <a class="logo" href="index.html"><img class="logo__icon" src="assets/figma/logo.svg" width="31" height="36" alt="">developer.dev</a>
      <details class="menu">
        <summary class="menu__toggle" aria-label="Open navigation"><span class="menu__line"></span><span class="menu__line"></span><span class="menu__line"></span></summary>
        <nav class="menu__nav" aria-label="Main navigation"><ul class="menu__list">
          <li><a href="index.html">Homepage</a></li><li><a href="about.html">About me</a></li>
          <li><a href="work.html">My work</a></li><li><a href="contact.html">Contact us</a></li>
        </ul></nav>
      </details>
    </header>`
    $('.site').prepend(header)
  }
  const menu = $('.menu__list')
  if (menu) {
    menu.insertAdjacentHTML('beforeend', `<li><a href="services.html">Services</a></li>
      <li><a href="cart.html">Cart (<span id="cart-count">0</span>)</a></li>`)
    const currentPage = location.pathname.split('/').pop() || 'index.html'
    $$('a', menu).forEach(link => {
      if (link.getAttribute('href') === currentPage) link.setAttribute('aria-current', 'page')
    })
  }
  $$('a[href="contact.html#demo-status"]').forEach(link => {
    link.href = 'privacy.html'
    link.target = '_blank'
    link.rel = 'noopener'
  })
}

async function renderCatalog() {
  const params = new URLSearchParams(new FormData($('#search-form')))
  const products = await api(`/api/products?${params}`)
  $('#products').innerHTML = products.map(product => `<article class="app-card">
    <a href="product.html?id=${product.id}"><img src="${escapeHTML(product.image)}" alt="${escapeHTML(product.name)}" loading="lazy"><h2>${escapeHTML(product.name)}</h2></a>
    <p class="app-muted">${escapeHTML(product.category)}</p><p>${escapeHTML(product.short_description)}</p>
    <div class="app-card__bottom">
      <p class="app-price">${money(product.price)}</p>
      <div class="app-card__actions">
        <a class="button" href="product.html?id=${product.id}">Подробнее</a>
        <button class="button" type="button" data-add="${product.id}">В корзину</button>
      </div>
    </div>
  </article>`).join('')
  status($('#page-status'), products.length ? `Найдено услуг: ${products.length}` : 'По вашему запросу ничего не найдено')
  history.replaceState(null, '', `${location.pathname}?${params}`)
}

async function renderProduct() {
  const id = Number(new URLSearchParams(location.search).get('id'))
  if (!Number.isSafeInteger(id) || id < 1) throw new Error('Услуга не выбрана')
  const product = await api(`/api/products/${id}`)
  document.title = `${product.meta_title || product.name} | developer.dev`
  $('#product-detail').innerHTML = `<div class="app-product">
    <img src="${escapeHTML(product.image)}" alt="${escapeHTML(product.name)}">
    <div><h1>${escapeHTML(product.name)}</h1><p class="app-muted">${escapeHTML(product.category)}</p>
      <p>${escapeHTML(product.description)}</p><p class="app-price">${money(product.price)}</p>
      <button type="button" class="button" data-add="${product.id}" ${product.available ? '' : 'disabled'}>${product.available ? 'В корзину' : 'Временно недоступна'}</button>
      <ul class="app-list">${(product.properties || []).map(prop => `<li>${escapeHTML(prop.property_name)}: ${escapeHTML(prop.property_value)}</li>`).join('')}</ul>
    </div></div><div class="app-gallery">${(product.images || []).map(image => `<figure><img src="${escapeHTML(image.image)}" alt="${escapeHTML(image.title)}"><figcaption>${escapeHTML(image.title)}</figcaption></figure>`).join('')}</div>`
}

function renderCart() {
  $('#cart-content').innerHTML = state.cart.items.length ? `<table class="app-table"><caption>Выбранные услуги</caption><thead><tr><th>Услуга</th><th>Цена</th><th>Количество</th><th>Сумма</th><th>Действия</th></tr></thead><tbody>
    ${state.cart.items.map(item => `<tr><td><a href="product.html?id=${item.product_id}">${escapeHTML(item.name)}</a></td>
      <td>${money(item.price)}</td><td><input type="number" min="1" max="${maxCartQuantity}" step="1" value="${item.quantity}" data-quantity="${item.product_id}" aria-label="Количество: ${escapeHTML(item.name)}"></td>
      <td>${money(item.price * item.quantity)}</td><td><button class="button" type="button" data-remove="${item.product_id}">Удалить</button></td></tr>`).join('')}
    </tbody></table>` : '<p>Ваша корзина пуста.</p>'
  $('#cart-total').textContent = `Итого: ${money(state.cart.total)}`
}

function bindFeedbackForm() {
  const form = $('[data-form="feedback"]')
  if (!form) return
  form.addEventListener('submit', async event => {
    event.preventDefault()
    const button = $('button[type="submit"]', form)
    if (button.disabled) return
    button.disabled = true
    const data = Object.fromEntries(new FormData(form))
    status($('[data-status]', form), 'Сохранение...')
    try {
      await api('/api/feedback', 'POST', { ...data, rating: data.rating ? Number(data.rating) : null, consent: form.elements.consent.checked })
      form.reset()
      status($('[data-status]', form), 'Обращение сохранено. Спасибо за сообщение!')
    } catch (error) {
      status($('[data-status]', form), error.message, true)
      form.elements[error.field]?.focus()
    } finally {
      button.disabled = false
    }
  })
}

async function onClick(event) {
  const button = event.target.closest('button')
  if (!button || button.disabled || state.busy) return
  if (!button.matches('[data-add], [data-remove], #clear-cart')) return
  state.busy = true
  button.disabled = true
  try {
    if ('add' in button.dataset) await addToCart(Number(button.dataset.add))
    if ('remove' in button.dataset) await saveCart(compactCart().filter(item => item.product_id !== Number(button.dataset.remove)))
    if (button.id === 'clear-cart') await saveCart([])
  } catch (error) {
    toast(error.message)
  } finally {
    state.busy = false
    button.disabled = false
  }
}

async function onChange(event) {
  const input = event.target
  if (!input.matches('[data-quantity]') || state.busy) return
  state.busy = true
  input.disabled = true
  try {
    const quantity = Number(input.value)
    if (!Number.isInteger(quantity) || quantity < 1 || quantity > maxCartQuantity) throw new Error(`Количество должно быть от 1 до ${maxCartQuantity}`)
    await saveCart(compactCart().map(item => item.product_id === Number(input.dataset.quantity) ? { ...item, quantity } : item))
  } catch (error) {
    toast(error.message)
    if ($('#cart-content')) renderCart()
  } finally {
    state.busy = false
    input.disabled = false
  }
}

async function init() {
  renderNavigation()
  bindFeedbackForm()
  document.addEventListener('click', onClick)
  document.addEventListener('change', onChange)
  try {
    await loadCart()
    if ($('#search-form')) {
      const categories = await api('/api/categories')
      $('#category-filter').insertAdjacentHTML('beforeend', categories.map(item => `<option value="${item.id}">${escapeHTML(item.name)}</option>`).join(''))
      const params = new URLSearchParams(location.search)
      for (const key of ['q', 'category', 'sort']) {
        if (params.has(key)) $('#search-form').elements[key].value = params.get(key)
      }
      $('#search-form').addEventListener('submit', event => {
        event.preventDefault()
        renderCatalog().catch(error => status($('#page-status'), error.message, true))
      })
      await renderCatalog()
    }
    if ($('#product-detail')) await renderProduct()
    if ($('#cart-content')) renderCart()
  } catch (error) {
    status($('#page-status'), error.message, true)
    toast(error.message)
  }
}

init()
