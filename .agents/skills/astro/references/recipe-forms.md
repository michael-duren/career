---
title: Forms Recipe
priority: medium
category: examples
---

# Recipe: Multi-Step Form with React

Interactive form with client-side state management and API integration.

## Form Component (React Island)

```jsx
// src/components/MultiStepForm.jsx
import { useState } from 'react';

export default function MultiStepForm() {
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    company: '',
    message: '',
  });
  const [status, setStatus] = useState('');

  const handleChange = (e) => {
    setFormData(prev => ({
      ...prev,
      [e.target.name]: e.target.value,
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setStatus('Sending...');

    try {
      const response = await fetch('/api/contact', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });

      if (response.ok) {
        setStatus('Message sent successfully!');
        setFormData({ name: '', email: '', company: '', message: '' });
        setStep(1);
      } else {
        setStatus('Error sending message.');
      }
    } catch (error) {
      setStatus('Network error.');
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      {step === 1 && (
        <div>
          <h2>Step 1: Your Info</h2>
          <input
            type="text"
            name="name"
            value={formData.name}
            onChange={handleChange}
            placeholder="Name"
            required
          />
          <input
            type="email"
            name="email"
            value={formData.email}
            onChange={handleChange}
            placeholder="Email"
            required
          />
          <button type="button" onClick={() => setStep(2)}>
            Next
          </button>
        </div>
      )}

      {step === 2 && (
        <div>
          <h2>Step 2: Company Info</h2>
          <input
            type="text"
            name="company"
            value={formData.company}
            onChange={handleChange}
            placeholder="Company"
          />
          <button type="button" onClick={() => setStep(1)}>
            Back
          </button>
          <button type="button" onClick={() => setStep(3)}>
            Next
          </button>
        </div>
      )}

      {step === 3 && (
        <div>
          <h2>Step 3: Message</h2>
          <textarea
            name="message"
            value={formData.message}
            onChange={handleChange}
            placeholder="Your message"
            required
          />
          <button type="button" onClick={() => setStep(2)}>
            Back
          </button>
          <button type="submit">Submit</button>
        </div>
      )}

      {status && <p>{status}</p>}
    </form>
  );
}
```

---

## API Endpoint

```typescript
// src/pages/api/contact.json.ts
import type { APIRoute } from 'astro';

export const POST: APIRoute = async ({ request }) => {
  const data = await request.json();

  // Validation
  if (!data.name || !data.email || !data.message) {
    return new Response(
      JSON.stringify({ error: 'Missing required fields' }),
      { status: 400 }
    );
  }

  // Send email (example with Resend)
  try {
    await fetch('https://api.resend.com/emails', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${import.meta.env.RESEND_API_KEY}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        from: 'noreply@example.com',
        to: 'hello@example.com',
        subject: `New contact from ${data.name}`,
        html: `
          <p><strong>Name:</strong> ${data.name}</p>
          <p><strong>Email:</strong> ${data.email}</p>
          <p><strong>Company:</strong> ${data.company || 'N/A'}</p>
          <p><strong>Message:</strong></p>
          <p>${data.message}</p>
        `,
      }),
    });

    return new Response(JSON.stringify({ success: true }), { status: 200 });
  } catch (error) {
    console.error(error);
    return new Response(
      JSON.stringify({ error: 'Failed to send email' }),
      { status: 500 }
    );
  }
};
```

---

## Page Integration

```astro
---
// src/pages/contact.astro
import Layout from '../layouts/Layout.astro';
import MultiStepForm from '../components/MultiStepForm.jsx';
---

<Layout title="Contact Us">
  <h1>Contact Us</h1>
  <MultiStepForm client:idle />
</Layout>
```

---

## Progressive Enhancement Alternative

```astro
---
// src/components/ContactForm.astro
---

<form id="contact-form" action="/api/contact" method="POST">
  <input type="text" name="name" required />
  <input type="email" name="email" required />
  <textarea name="message" required></textarea>
  <button type="submit">Send</button>
</form>

<script>
  // Progressive enhancement (works without JS too)
  const form = document.getElementById('contact-form') as HTMLFormElement;
  
  form?.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const formData = new FormData(form);
    const data = Object.fromEntries(formData);
    
    try {
      const response = await fetch('/api/contact', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      
      if (response.ok) {
        alert('Message sent!');
        form.reset();
      } else {
        alert('Error sending message');
      }
    } catch (error) {
      console.error(error);
      // Fallback: submit form normally
      form.submit();
    }
  });
</script>
```

---

## Resources

- See [api-endpoints.md](api-endpoints.md) for API patterns
