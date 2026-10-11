# frozen_string_literal: true

module Ledger
  module Reporting
    MONTH_NAMES = %w[Jan Feb Mar Apr May Jun Jul Aug Sep Oct Nov Dec].freeze

    def self.monthly_summary(account, year)
      account.entries.group_by { |entry| entry.posted_on.month }.map do |month, entries|
        [MONTH_NAMES[month - 1], year, entries.sum(&:signed_amount)]
      end
    end

    def self.format_cents(cents, currency)
      format("%<amount>.2f %<currency>s", amount: cents / 100.0, currency: currency)
    end
  end
end
